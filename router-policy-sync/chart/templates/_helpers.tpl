{{- define "router-policy-sync.name" -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}