{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "redis.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "redis.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Return the appropriate apiVersion for networkpolicy.
*/}}
{{- define "networkPolicy.apiVersion" -}}
{{- if semverCompare ">=1.4-0, <1.7-0" .Capabilities.KubeVersion.GitVersion -}}
{{- print "extensions/v1beta1" -}}
{{- else if semverCompare "^1.7-0" .Capabilities.KubeVersion.GitVersion -}}
{{- print "networking.k8s.io/v1" -}}
{{- end -}}
{{- end -}}


{{/*
Calculates Existing pvc name
*/}}
{{- define "redisStore.existingPvc" -}}
{{- $existingPvc := coalesce .Values.existingPvc .Values.existingClaim .Values.pvcName .Values.persistence.existingClaim | default "" -}}
{{- printf "%s" $existingPvc -}}
{{- end -}}

{{/*
Calculates pvcName
*/}}
{{- define "redisStore.pvcName" -}}
{{- $pvcName := include "redisStore.existingPvc" .  | default (include "redis.fullname" . ) -}}
{{- printf "%s" $pvcName -}}
{{- end -}}

{{/*
Calculates storage class name
*/}}
{{- define "redisStore.storageClass" -}}
{{- $storageClass := coalesce .Values.storageClass .Values.StorageClass .Values.persistence.storageClass .Values.global.storageClass | default "" -}}
{{- printf "%s" $storageClass -}}
{{- end -}}

{{/*
Calculates storage size
*/}}

{{- define "redisStore.storageSize" -}}
{{- $storageSize := coalesce .Values.storageSize .Values.persistence.size -}}
{{- printf "%s" $storageSize -}}
{{- end -}}