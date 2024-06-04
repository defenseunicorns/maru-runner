#!/usr/bin/env sh

# Create the json schema for tasks.yaml
go run main.go internal config-tasks-schema > tasks.schema.json

# Adds pattern properties to all definitions to allow for yaml extensions
jq '
  def addPatternProperties:
    . +
    if has("properties") then
      {"patternProperties": {"^x-": {}}}
    else
      {}
    end;

  walk(if type == "object" then addPatternProperties else . end)
' tasks.schema.json > temp_tasks.schema.json

mv temp_tasks.schema.json tasks.schema.json

awk '{gsub(/\[github\.com\/defenseunicorns\/maru-runner\/src\/pkg\/variables\.ExtraVariableInfo\]/, ""); print}' tasks.schema.json > temp_tasks.schema.json

mv temp_tasks.schema.json tasks.schema.json
