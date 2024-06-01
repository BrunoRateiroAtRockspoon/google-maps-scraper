module github.com/gosom/google-maps-scraper

go 1.20

require (
	github.com/PuerkitoBio/goquery v1.9.2
	github.com/google/uuid v1.6.0
	github.com/gosom/scrapemate v0.6.0
	github.com/jackc/pgx/v5 v5.5.5
	github.com/mcnijman/go-emailaddress v1.1.1
	github.com/playwright-community/playwright-go v0.4201.1
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.9.0
)

replace github.com/gosom/scrapemate => ./local-scrapemate

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
)

require (
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/deckarep/golang-set/v2 v2.6.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.4 // indirect
	github.com/go-jose/go-jose/v3 v3.0.3 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.20.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gosom/kit v0.0.0-20230309082109-543b32ac686a // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rs/zerolog v1.32.0 // indirect
	github.com/syndtr/goleveldb v1.0.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/exp v0.0.0-20240416160154-fe59bbe5cc7f // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

//replace github.com/gosom/scrapemate v0.6.0 => ../scrapemate
