{{- define "mdns-reflector.name" -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "mdns-reflector.serviceAccountName" -}}
{{ default (include "mdns-reflector.name" .) .Values.serviceAccount.name }}
{{- end }}
