{{- define "domain-expiry-exporter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "domain-expiry-exporter.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "domain-expiry-exporter.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "domain-expiry-exporter.labels" -}}
helm.sh/chart: {{ include "domain-expiry-exporter.chart" . }}
{{ include "domain-expiry-exporter.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "domain-expiry-exporter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "domain-expiry-exporter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "domain-expiry-exporter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "domain-expiry-exporter.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "domain-expiry-exporter.redisEnabled" -}}
{{- if or .Values.redis.enabled .Values.valkey.enabled -}}true{{- end -}}
{{- end }}

{{- define "domain-expiry-exporter.redisHost" -}}
{{- if .Values.redis.host -}}
{{- .Values.redis.host -}}
{{- else if .Values.valkey.fullnameOverride -}}
{{- .Values.valkey.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default "valkey" .Values.valkey.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end }}

{{- define "domain-expiry-exporter.redisSecretName" -}}
{{- if .Values.redis.passwordSecret.name -}}
{{- .Values.redis.passwordSecret.name -}}
{{- else if .Values.valkey.enabled -}}
{{- .Values.valkey.auth.usersExistingSecret -}}
{{- end -}}
{{- end }}
