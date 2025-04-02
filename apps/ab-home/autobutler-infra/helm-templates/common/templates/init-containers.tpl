{{- define "chart.initContainers" -}}
{{- range .Values.initContainers }}
- name: {{ .name }}
  image: {{ .image }}
  command: {{ .command }}
    {{- if .extraArgs }}
    args:
      {{- range .extraArgs }}
      - "{{ . }}"
      {{- end }}
    {{- end }}
  volumeMounts:
  {{- range .volumeMounts }}
    - name: {{ .name }}
      mountPath: {{ .mountPath }}
      {{- if .subPath }}
      subPath: {{ .subPath }}
      {{- end }}
  {{- end }}
{{- end }}
{{- end }}