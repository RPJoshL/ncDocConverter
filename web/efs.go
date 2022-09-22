package web

import (
    "embed"
)

//go:embed "app/dist"
var FrontendFiles embed.FS

//go:embed "app/src"
var DevelopeFiles embed.FS

//go:embed "template"
var TemplateFiles embed.FS