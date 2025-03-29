package config

const (
    Version       = "0.2.0"
    Slogan        = "\n     _    _\n    | |  / )\n    | | / /_   _ ____\n    | |< <| | | |  _ \\\n    | | \\ \\ |_| | | | |\n    |_|  \\_)____|_| |_|\n\n A CLI tool for building golang application. \n"
    WireCmd       = "github.com/google/wire/cmd/wire@latest"
    KunCmd        = "github.com/spruce1698/kun@latest"
    RunExcludeDir = ".git,.idea,tmp,vendor"
    RunIncludeExt = "go,html,yaml,yml,toml,ini,json,xml,tpl,tmpl"
    Short         = Slogan + " Kun " + Version + " - Copyright (c) 2025 spruce1698\n Released under the MIT License.\n \n"
)
