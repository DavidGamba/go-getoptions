#!/usr/bin/env bash

go build \
	-ldflags "-X \"github.com/DavidGamba/go-getoptions/text.ErrorMissingRequiredOption=falta opción requerida '%s'\"
	-X \"github.com/DavidGamba/go-getoptions/text.HelpNameHeader=NOMBRE\"
	-X \"github.com/DavidGamba/go-getoptions/text.HelpSynopsisHeader=SINOPSIS\"
	-X \"github.com/DavidGamba/go-getoptions/text.HelpCommandsHeader=COMANDOS\"
	-X \"github.com/DavidGamba/go-getoptions/text.HelpRequiredOptionsHeader=PARAMETROS REQUERIDOS\"
	-X \"github.com/DavidGamba/go-getoptions/text.HelpOptionsHeader=OPCIONES\""
