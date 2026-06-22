{{- define "gateway-route-sync.name" -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}