# A SonarQube to Redmine issues  importer

Import SonarQube issues to Redmine

### Usage

##### Using pre-built binaries

See [releases](https://github.com/pierDipi/sonarqube-redmine-issues-importer/releases)

```bash
sonarqube-redmine-issues-importer \
    --sonarqube-issues-search-url="<your_sonarqube_api_issue_search_url>" \
    --redmine-base-url="<redmine_url>" \
    --redmine-api-key="<your_api_key>" \
    --redmine-project-id="<project_id>" \
    --redmine-tracker-id="<tracker_id>" \
    --redmine-parent-issue-id="<redmine_parent_issue_id>"
```

##### Using source code

Prerequisites:
- git
- go (golang)

Clone the repository:
- https: `git clone https://github.com/pierDipi/sonarqube-redmine-issues-importer.git`
- ssh: `git clone git@github.com:pierDipi/sonarqube-redmine-issues-importer.git`

Run 
```bash
cd sonarqube-redmine-issues-importer/sonarqube-redmine-issues-importer
go run main.go \
    --sonarqube-issues-search-url="<your_sonarqube_api_issue_search_url>" \
    --redmine-base-url="<redmine_url>" \
    --redmine-api-key="<your_api_key>" \
    --redmine-project-id="<project_id>" \
    --redmine-tracker-id="<tracker_id>" \
    --redmine-parent-issue-id="<redmine_parent_issue_id>"
```
