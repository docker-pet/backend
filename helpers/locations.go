package helpers

import "github.com/biter777/countries"

func GetCountryCodes() []string {
	var codes []string
	for _, country := range countries.All() {
		codes = append(codes, country.Alpha2())
	}
	return codes
}
