package web

const repositoriesHtmlTemplate = `
<!DOCTYPE html>
<html>
<head>
  <style type="text/css">
.hidden {
    display: none
}
pre.tags {
    font-size: larger;
    font-weight: bold
}
pre {
    margin: 4px;
}
li.image {
    border: 1px solid #aaaaaa;
    background: #fffff0;
    padding: 4px;
    margin-bottom: 8px;
}
li.list {
    border: 1px dashed #aaaaaa;
    background: #fffff8;
    padding: 4px;
    margin-bottom: 8px;
}
li {
    list-style-type: none;
}
li.list ul {
   padding: 0px;
}
ul {
    padding-left: 0px;
}
  </style>
  <script>
    function toggleDetails(event) {
        details = event.currentTarget.querySelector(".details").classList.toggle("hidden")
    }
  </script>
</head>
<body>
{{define "Image" -}}
digest: {{.Digest}}
mediaType: {{.MediaType}}
{{- with .Title }}
title: {{.}}
{{- end }}
{{- with .Description }}
description: {{.}}
{{- end }}
architecture: {{.Architecture}}
os: {{.OS}}
{{- with .Annotations}}
annotations:
{{- range $k, $v := .}}
    {{$k}}: {{$v}}
{{- end}}
{{- end -}}
{{- with .Labels}}
labels:
{{- range $k, $v := .}}
    {{$k}}: {{$v}}
{{- end}}
{{- end -}}
{{- end}}
<ul>
{{- range .}}
<li>
<h2>{{.Name}}</h2>
<ul>
{{- range .Images}}
<li class="image" onclick="toggleDetails(event)">
<pre class="tags">{{- range .Tags}}{{ . }} {{- end }}</pre>
<pre class="details {{if .IsLatest}}{{else}}hidden{{end}}">{{template "Image" .}}</pre>
</li>
{{end}}
{{- range .Lists}}
<li class="list">
<ul>
<pre class="tags">{{- range .Tags}}{{ . }} {{- end }}</pre>
{{- range .Images}}
<li class="image" onclick="toggleDetails(event)">
<pre>{{template "Image" .}}</pre>
</li>
{{end}}
</ul>
<li>
{{end}}
</ul>
</li>
{{end}}
</ul>
</body>
`
