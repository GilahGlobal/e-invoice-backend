package resources

import "einvoice-access-point/internal/data/entities"

type ResourceItemDto struct {
	Code  string `json:"code" example:"380"`
	Value string `json:"value" example:"Credit Note"`
}

type ResourcesResponseDto struct {
	entities.Response
	Data []ResourceItemDto `json:"data"`
}

type HSNCodesItemDto struct {
	HSCode      string `json:"hscode" example:"0101.21"`
	Description string `json:"description" example:"Horses; live, pure-bred breeding animals"`
}

type HSNCodesResponseDto struct {
	entities.Response
	Data []HSNCodesItemDto `json:"data"`
}

type ServiceCodesItemDto struct {
	Code        string `json:"code" example:"0111"`
	Description string `json:"description" example:"Growing of cereals (except rice), leguminous crops and oil seeds"`
}

type ServiceCodesResponseDto struct {
	entities.Response
	Data []ServiceCodesItemDto `json:"data"`
}

type TaxCategoryItemDto struct {
	Code    string `json:"code" example:"STANDARD_GST"`
	Value   string `json:"value" example:"Standard Goods and Services Tax"`
	Percent string `json:"percent" example:"Not Available"`
}

type TaxCategoriesResponseDto struct {
	entities.Response
	Data []TaxCategoryItemDto `json:"data"`
}

type CountryItemDto struct {
	Name                   string `json:"name" example:"Afghanistan"`
	Alpha2                 string `json:"alpha_2" example:"AF"`
	Alpha3                 string `json:"alpha_3" example:"AFG"`
	CountryCode            string `json:"country_code" example:"004"`
	Iso31662               string `json:"iso_3166_2" example:"ISO 3166-2:AF"`
	Region                 string `json:"region" example:"Asia"`
	SubRegion              string `json:"sub_region" example:"Southern Asia"`
	IntermediateRegion     string `json:"intermediate_region" example:""`
	RegionCode             string `json:"region_code" example:"142"`
	SubRegionCode          string `json:"sub_region_code" example:"034"`
	IntermediateRegionCode string `json:"intermediate_region_code" example:""`
}

type CountriesResponseDto struct {
	entities.Response
	Data []CountryItemDto `json:"data"`
}

type CurrencyItemDto struct {
	Symbol        string `json:"symbol" example:"$"`
	Name          string `json:"name" example:"US Dollar"`
	SymbolNative  string `json:"symbol_native" example:"$"`
	DecimalDigits int    `json:"decimal_digits" example:"2"`
	Rounding      int    `json:"rounding" example:"0"`
	Code          string `json:"code" example:"USD"`
	NamePlural    string `json:"name_plural" example:"US dollars"`
}

type CurrenciesResponseDto struct {
	entities.Response
	Data []CurrencyItemDto `json:"data"`
}

type LGAItemDto struct {
	Name      string `json:"name" example:"Aba North"`
	Code      string `json:"code" example:"NG-AB-ANO"`
	StateCode string `json:"state_code" example:"NG-AB"`
}

type LGAsResponseDto struct {
	entities.Response
	Data []LGAItemDto `json:"data"`
}

type StateItemDto struct {
	Name string `json:"name" example:"Abia"`
	Code string `json:"code" example:"NG-AB"`
}

type StatesResponseDto struct {
	entities.Response
	Data []StateItemDto `json:"data"`
}
