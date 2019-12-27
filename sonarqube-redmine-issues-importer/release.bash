#!/bin/bash

./cross_compile.bash

for filename in sonarqube-redmine-issues-importer-*; do
    zip -r "$filename.zip" "$filename"
    rm "$filename"
done