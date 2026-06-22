{{- define "dynamic-dns.name" -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "dynamic-dns.serviceAccountName" -}}
{{ default (include "dynamic-dns.name" .) .Values.serviceAccount.name }}
{{- end }}