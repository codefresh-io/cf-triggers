{{/*
We create Deployment resource as template to be able to use many deployments but with
different name and version. This is for Istio POC.
*/}}
{{- define "hermes.renderDeployment" -}}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ template "hermes.fullname" $ }}-{{ .version | default "base" }}
  labels:
    app: {{ template "hermes.fullname" . }}
    role: {{ template "hermes.role" . }}
    chart: "{{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}"
    release: {{ .Release.Name  | quote }}
    heritage: {{ .Release.Service  | quote }}
    version: {{ .version | default "base" | quote  }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      app: {{ template "hermes.name" . }}
  template:
    metadata:
      annotations:
        checksum/config: {{ include (print .Template.BasePath "/configmap.yaml") . | sha256sum }}
        sidecar.istio.io/inject: {{ $.Values.global.istio.enabled | default "false" | quote }}
      labels:
        app: {{ template "hermes.name" . }}
        role: {{ template "hermes.role" . }}
        release: {{ .Release.Name }}
        heritage: {{ .Release.Service  | quote }}
        version: {{ .version | default "base" | quote  }}
    spec:
      {{- if not .Values.global.devEnvironment }}
      {{- $podSecurityContext := (kindIs "invalid" .Values.global.podSecurityContextOverride) | ternary .Values.podSecurityContext .Values.global.podSecurityContextOverride }}
      {{- with $podSecurityContext }}
      securityContext:
{{ toYaml . | indent 8}}
      {{- end }}
      {{- end }}
      volumes:
        - name: config-volume
          configMap:
            name: {{ template "hermes.fullname" . }}
      imagePullSecrets:
        - name: "{{ .Release.Name }}-{{ .Values.global.codefresh }}-registry"
      containers:
        - name: {{ .Chart.Name }}
          {{- if .Values.global.privateRegistry }}
          image: "{{ .Values.global.dockerRegistry }}{{ .Values.image.name }}:{{ .imageTag }}"
          {{- else }}
          image: "{{ .Values.image.dockerRegistry }}{{ .Values.image.name }}:{{ .imageTag }}"
          {{- end }}
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - containerPort: {{ .Values.service.internalPort }}
          env:
          {{- if .Values.global.env }}
          {{- range $key, $value := .Values.global.env }}
          - name: {{ $key }}
            value: {{ $value | quote }}
          {{- end}}
          {{- end}}
          - name: LOG_LEVEL
            value: {{ .Values.logLevel | quote }}
          - name: LOG_JSON
            value: {{ .Values.logJSON | quote }}
          {{- if ne .Values.logLevel "debug" }}
          - name: GIN_MODE
            value: release
          {{- end }}
          - name: NEWRELIC_LICENSE_KEY
            valueFrom:
              secretKeyRef:
                name: "{{ .Release.Name }}-{{ .Values.global.codefresh }}"
                key: newrelic-license-key
          {{- if .Values.redis.enabled }}  
          - name: STORE_HOST
            value: "{{ .Release.Name }}-{{ .Values.redis.nameOverride }}"
          - name: STORE_PORT
            value: {{ .Values.redis.port | quote }}
          - name: STORE_DB
            value: "{{ .Values.redis.db }}"
          - name: STORE_PASSWORD
            valueFrom:
              secretKeyRef:
                name: "{{ .Release.Name }}-{{ .Values.redis.nameOverride }}"
                key: redis-password
          {{- else }}
          - name: STORE_HOST
            value: {{ default (printf "%s-%s" $.Release.Name $.Values.global.redisService) $.Values.global.redisUrl }}
          - name: STORE_PORT
            value: {{ .Values.redis.port | quote }}
          - name: STORE_DB
            value: "{{ .Values.redis.db }}"
          - name: STORE_PASSWORD
            valueFrom:
              secretKeyRef:
                name: "{{ $.Release.Name }}-{{ $.Values.global.codefresh }}"
                key: redis-password
          {{- end }}
          - name: TYPES_CONFIG
            value: {{ default "/etc/hermes/type_config.json" .Values.types_config }}
          - name: PORT
            value: {{ .Values.service.internalPort | quote }}
          - name: CFAPI_URL
            {{- if .Values.global.cfapiService }}
            value: "{{.Values.cfapi.protocol}}://{{.Release.Name}}-{{.Values.global.cfapiService}}:{{.Values.global.cfapiInternalPort}}"
            {{- else }}
            value: "{{.Values.cfapi.protocol}}://{{.Release.Name}}-cfapi:{{.Values.cfapi.port}}"
            {{- end }}
          - name: CFAPI_TOKEN
            valueFrom:
              secretKeyRef:
                name: {{ template "hermes.fullname" . }}
                key: cfapi-token
          volumeMounts:
            - name: config-volume
              mountPath: /etc/hermes
          livenessProbe:
            httpGet:
              path: /health
              port: {{ .Values.service.internalPort }}
            initialDelaySeconds: 5
            periodSeconds: 15
            timeoutSeconds: 3
            failureThreshold: 5
          readinessProbe:
            httpGet:
              path: /ping
              port: {{ .Values.service.internalPort }}
            initialDelaySeconds: 5
            timeoutSeconds: 3
            periodSeconds: 15
            failureThreshold: 5
          resources:
{{ toYaml .Values.resources | indent 12 }}
    {{- if .Values.nodeSelector }}
      nodeSelector:
{{ toYaml .Values.nodeSelector | indent 8 }}
    {{- end }}
{{- with (default $.Values.global.appServiceTolerations $.Values.tolerations ) }}
      tolerations:
{{ toYaml . | indent 8}}
      {{- end }}
      affinity:
{{ toYaml (default .Values.global.appServiceAffinity .Values.affinity) | indent 8 }}
{{- end }}
