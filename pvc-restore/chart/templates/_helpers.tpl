{{- define "pvc-restore.name" -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}