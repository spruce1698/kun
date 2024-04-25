package tpl

import "embed"

//go:embed create/*.tpl
var CreateTplFS embed.FS

//go:embed basic.zip advanced.zip
var NewTplZipFS embed.FS

//
// //go:embed basic/* advanced/*
// var NewTplDirFS embed.FS
