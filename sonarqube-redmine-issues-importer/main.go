package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type SonarqubeTextRange struct {
	StartLine   uint64 `json:"startLine"`
	EndLine     uint64 `json:"endLine"`
	StartOffset uint64 `json:"startOffset"`
	EndOffset   uint64 `json:"endOffset"`
}

type SonarqubeLocation struct {
	Component string             `json:"component"`
	TextRange SonarqubeTextRange `json:"textRange"`
	Message   string             `json:"msg"`
}

type SonarqubeFlow struct {
	Locations []SonarqubeLocation `json:"locations"`
}

type SonarqubeIssue struct {
	Key          string             `json:"key"`
	Rule         string             `json:"rule"`
	Severity     string             `json:"severity"`
	Component    string             `json:"component"`
	Project      string             `json:"project"`
	Line         uint64             `json:"line"`
	Hash         string             `json:"hash"`
	TextRange    SonarqubeTextRange `json:"textRange"`
	Flows        []SonarqubeFlow    `json:"flows"`
	Status       string             `json:"status"`
	Message      string             `json:"message"`
	Effort       string             `json:"effort"`
	Debt         string             `json:"debt"`
	Tags         []string           `json:"tags"`
	CreationDate string             `json:"creationDate"`
	UpdateDate   string             `json:"updateDate"`
	Type         string             `json:"type"`
}

type SonarqubePaging struct {
	PageIndex uint64 `json:"pageIndex"`
	PageSize  uint64 `json:"pageSize"`
	Total     uint64 `json:"total"`
}

type SonarqubeResponse struct {
	Paging SonarqubePaging  `json:"paging"`
	Issues []SonarqubeIssue `json:"issues"`
}

type RedmineIssue struct {
	ProjectId      string  `json:"project_id"`
	TrackerId      string  `json:"tracker_id"`
	StatusId       string  `json:"status_id"`
	PriorityId     string  `json:"priority_id"`
	Subject        string  `json:"subject"`
	Description    string  `json:"description"`
	ParentIssueId  string  `json:"parent_issue_id"`
	EstimatedHours float64 `json:"estimated_hours"`

	CustomFields []struct{
		Id string `json:"id"`
		Value string `json:"value"`
	} `json:"custom_fields"`
}

type RedmineRequest struct {
	Issue RedmineIssue `json:"issue"`
}

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type redmineFlags struct {
	URL           string
	apiKey        string
	projectId     string
	trackerId     string
	parentIssueId string
}

func run() error {

	sonarqubeIssueSearchURL := "localhost:9000/api/issues/search"
	redmineFlags := redmineFlags{
		URL:           "",
		apiKey:        "",
		projectId:     "",
		trackerId:     "",
		parentIssueId: "",
	}

	flag.StringVar(&sonarqubeIssueSearchURL, "sonarqube-issues-search-url", sonarqubeIssueSearchURL, "Sonarqube issues search URL (without pageIndex query param)")
	flag.StringVar(&redmineFlags.URL, "redmine-base-url", redmineFlags.URL, "Redmine base URL")
	flag.StringVar(&redmineFlags.apiKey, "redmine-api-key", redmineFlags.apiKey, "Redmine API key")
	flag.StringVar(&redmineFlags.projectId, "redmine-project-id", redmineFlags.projectId, "Redmine project identifier")
	flag.StringVar(&redmineFlags.trackerId, "redmine-tracker-id", redmineFlags.trackerId, "Redmine tracker identifier")
	flag.StringVar(&redmineFlags.parentIssueId, "redmine-parent-issue-id", redmineFlags.parentIssueId, "Redmine parent issue identifier")
	flag.Parse()

	sonarqubeResponse, err := getSonarqubeIssues(sonarqubeIssueSearchURL)
	if err != nil {
		return err
	}
	importIssues(sonarqubeResponse.Issues, redmineFlags)

	numPages := sonarqubeResponse.Paging.Total/sonarqubeResponse.Paging.PageSize + 1

	for i := 2; i <= int(numPages); i++ {
		issueSearchURL := sonarqubeIssueSearchURL + "&pageIndex=" + strconv.Itoa(i)
		response, err := getSonarqubeIssues(issueSearchURL)
		if err != nil {
			return err
		}
		importIssues(response.Issues, redmineFlags)
	}

	return nil
}

func importIssues(sonarqubeIssues []SonarqubeIssue, redmineFlags redmineFlags) {
	for i := range sonarqubeIssues {
		sonarqubeIssue := sonarqubeIssues[i]
		redmineIssue, err := transformToRedmineIssue(redmineFlags, sonarqubeIssue)
		if err != nil {
			log.Println(err)
			log.Println(sonarqubeIssue)
			continue
		}
		err = createRedmineIssue(redmineFlags, redmineIssue)
		if err != nil {
			log.Println(err)
			log.Println(sonarqubeIssue)
		}
		after := time.After(2 * time.Second)
		select {
		case <-after:
		}
	}
}

func getSonarqubeIssues(sonarqubeURLStr string) (SonarqubeResponse, error) {
	sonarqubeURL, err := url.Parse(sonarqubeURLStr)
	if err != nil {
		return SonarqubeResponse{}, fmt.Errorf("sonarqube url: %w", err)
	}

	sonarqubeResponse, err := http.Get(sonarqubeURL.String())
	if err != nil {
		return SonarqubeResponse{}, fmt.Errorf("sonarqube server: %w", err)
	}
	defer sonarqubeResponse.Body.Close()

	if sonarqubeResponse.StatusCode != http.StatusOK {
		return SonarqubeResponse{}, fmt.Errorf("sonarqube server response status code is not 200")
	}

	body, err := ioutil.ReadAll(sonarqubeResponse.Body)
	if err != nil {
		return SonarqubeResponse{}, fmt.Errorf("sonarqube response body: %w", err)
	}

	var sonarqubeResponseBody SonarqubeResponse
	if err := json.Unmarshal(body, &sonarqubeResponseBody); err != nil {
		return SonarqubeResponse{}, fmt.Errorf("sonarqube response body: %w", err)
	}

	return sonarqubeResponseBody, nil
}

func createRedmineIssue(redmineFlags redmineFlags, issue RedmineIssue) error {
	requestBody := RedmineRequest{Issue: issue}
	redmineIssueJson, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	var client http.Client
	request, err := http.NewRequest("POST", redmineFlags.URL+"/issues.json", bytes.NewBuffer(redmineIssueJson))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("X-Redmine-API-Key", redmineFlags.apiKey)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return fmt.Errorf("response status code %d", response.StatusCode)
	}
	return nil
}

func transformToRedmineIssue(redmineFlags redmineFlags, issue SonarqubeIssue) (RedmineIssue, error) {
	description, err := getRedmineDescription(issue)
	if err != nil {
		return RedmineIssue{}, err
	}

	subject, err := getRedmineSubject(issue)
	if err != nil {
		return RedmineIssue{}, err
	}

	duration, err := getDuration(issue.Debt)
	estimatedHours := 0.0
	if err == nil {
		estimatedHours = duration.Hours()
	} else {
		log.Println("cannot determine duration of the issue", issue)
	}

	return RedmineIssue{
		ProjectId:      redmineFlags.projectId,
		TrackerId:      redmineFlags.trackerId,
		StatusId:       "1",
		PriorityId:     "2",
		Subject:        subject,
		Description:    description,
		ParentIssueId:  redmineFlags.parentIssueId,
		EstimatedHours: estimatedHours,
	}, nil
}

func getRedmineSubject(issue SonarqubeIssue) (string, error) {

	funcMap := template.FuncMap{
		"getSquid": func(rule string) string {
			squidIndex := strings.LastIndex(issue.Rule, ":")
			squidBuf := bytes.NewBufferString("")
			if squidIndex >= 0 && squidIndex+1 < len(issue.Rule) {
				squidBuf.WriteString(issue.Rule[squidIndex+1:])
				squidBuf.WriteString(":")
			}
			return squidBuf.String()
		},
		"getFile": func(message string) string {
			fileIndex := strings.LastIndex(issue.Component, "/")
			subjectBuffer := bytes.NewBufferString("")
			if fileIndex >= 0 && fileIndex+1 < len(issue.Component) {
				subjectBuffer.WriteString(issue.Component[fileIndex+1:])
			}
			return subjectBuffer.String()
		},
	}

	tmplStr := "{{getSquid .Rule}} {{.Message}} - {{getFile .Component}}"

	tmpl, err := template.New("redmine_subject").
		Funcs(funcMap).
		Parse(tmplStr)

	if err != nil {
		return "", err
	}

	buf := bytes.NewBufferString("")
	if err := tmpl.Execute(buf, issue); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func getRedmineDescription(issue SonarqubeIssue) (string, error) {
	tmplStr := `
Message: {{.Message}}

Component: {{.Component}}
Line: {{.Line}}

Text range:
	- start line: {{.TextRange.StartLine}}
	- end line: {{.TextRange.EndLine}}
	- start offset {{.TextRange.StartOffset}}
	- end offset {{.TextRange.EndOffset}}	

Key: {{.Key}}
Rule: {{.Rule}}
Type: {{.Type}}
Debt: {{.Debt}}
Effort: {{.Effort}}

Tags: {{range .Tags}}
	- {{.}}{{end}}
`
	tmpl, err := template.New("redmine_description").
		Parse(tmplStr)

	if err != nil {
		return "", err
	}

	buf := bytes.NewBufferString("")
	if err := tmpl.Execute(buf, issue); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func getDuration(s string) (time.Duration, error) {
	if duration, err := time.ParseDuration(s); err == nil {
		return duration, nil
	}
	if len(s) > 2 {
		durationStr := s[:len(s)-2]
		effort, err := time.ParseDuration(durationStr)
		if err == nil {
			return effort, nil
		}
	}
	return time.Nanosecond, fmt.Errorf("cannot parse duration: %s", s)
}
