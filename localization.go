package mlx

import (
	"context"
	"strings"
)

// LocaleProfile bundles localization and timezone fingerprint for a country.
type LocaleProfile struct {
	Localization *LocalizationFingerprint
	Timezone     *TimezoneFingerprint
}

// LocaleForCountry returns a preset LocaleProfile for the given ISO 3166-1
// alpha-2 country code. If the country is unknown, it falls back to en-US / UTC.
func LocaleForCountry(countryCode string) *LocaleProfile {
	cc := strings.ToUpper(strings.TrimSpace(countryCode))
	if lp, ok := localeTable[cc]; ok {
		return lp
	}
	return localeTable["US"] // fallback
}

// PatchProfileForProxy installs the given proxy into a profile and automatically
// adjusts language, locale, timezone, screen, and browser UI language to match
// the proxy country. If the proxy country is not already known, it is resolved
// via the launcher ValidateProxy endpoint.
func (c *Client) PatchProfileForProxy(ctx context.Context, profileID string, proxy *Proxy) (*EmptyDataResponse, *Response, error) {
	if proxy == nil {
		return nil, nil, NewArgError("proxy", "it must not be nil")
	}
	if profileID == "" {
		return nil, nil, NewArgError("profileID", "it must not be empty")
	}

	countryCode, err := c.resolveProxyCountry(ctx, proxy)
	if err != nil {
		return nil, nil, err
	}

	locale := LocaleForCountry(countryCode)

	req := &PatchProfileRequest{
		ProfileID: profileID,
		Proxy:     proxy,
		Parameters: &ProfileParameters{
			Flags: &ProfileFlags{
				LocalizationMasking: "custom",
				TimezoneMasking:     "custom",
				ProxyMasking:        "custom",
				ScreenMasking:       "custom",
			},
			Fingerprint: &Fingerprint{
				Localization: locale.Localization,
				Timezone:     locale.Timezone,
				Screen:       &ScreenFingerprint{Width: 1920, Height: 1080, PixelRatio: 1.0},
				CMDParams: &CommandParams{
					Params: []CommandParam{
						{Flag: "--lang", Value: locale.Localization.Locale},
						{Flag: "--window-size", Value: "1920,1080"},
					},
				},
			},
		},
	}

	return c.Profiles.Patch(ctx, req)
}

// resolveProxyCountry determines the ISO country code for a proxy. It prefers
// proxy.Country, then falls back to launcher ValidateProxy, and finally "US".
func (c *Client) resolveProxyCountry(ctx context.Context, proxy *Proxy) (string, error) {
	if cc := strings.TrimSpace(proxy.Country); cc != "" {
		return cc, nil
	}

	// Try ValidateProxy as a fallback.
	if proxy.Host != "" && proxy.Port > 0 {
		validateReq := &ValidateProxyRequest{
			Type:     proxy.Type,
			Host:     proxy.Host,
			Port:     proxy.Port,
			Username: proxy.Username,
			Password: proxy.Password,
		}
		resp, _, err := c.Launcher.ValidateProxy(ctx, validateReq)
		if err == nil && resp != nil && resp.Data.CountryCode != "" {
			return resp.Data.CountryCode, nil
		}
		// Validation failed; proceed with default.
	}

	return "US", nil
}

// localeTable maps ISO 3166-1 alpha-2 country codes to LocaleProfile presets.
var localeTable = map[string]*LocaleProfile{
	// ── Americas ──────────────────────────────────────────────
	"US": {
		Localization: &LocalizationFingerprint{
			Languages:       "en-US",
			Locale:          "en-US",
			AcceptLanguages: "en-US,en;q=0.9",
		},
		Timezone: &TimezoneFingerprint{Zone: "America/New_York"},
	},
	"CA": {
		Localization: &LocalizationFingerprint{
			Languages:       "en-CA",
			Locale:          "en-CA",
			AcceptLanguages: "en-CA,en;q=0.9,fr-CA;q=0.7,fr;q=0.5",
		},
		Timezone: &TimezoneFingerprint{Zone: "America/Toronto"},
	},
	"BR": {
		Localization: &LocalizationFingerprint{
			Languages:       "pt-BR",
			Locale:          "pt-BR",
			AcceptLanguages: "pt-BR,pt;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "America/Sao_Paulo"},
	},
	"MX": {
		Localization: &LocalizationFingerprint{
			Languages:       "es-MX",
			Locale:          "es-MX",
			AcceptLanguages: "es-MX,es;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "America/Mexico_City"},
	},
	"AR": {
		Localization: &LocalizationFingerprint{
			Languages:       "es-AR",
			Locale:          "es-AR",
			AcceptLanguages: "es-AR,es;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "America/Buenos_Aires"},
	},
	"CO": {
		Localization: &LocalizationFingerprint{
			Languages:       "es-CO",
			Locale:          "es-CO",
			AcceptLanguages: "es-CO,es;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "America/Bogota"},
	},
	"CL": {
		Localization: &LocalizationFingerprint{
			Languages:       "es-CL",
			Locale:          "es-CL",
			AcceptLanguages: "es-CL,es;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "America/Santiago"},
	},
	"AU": {
		Localization: &LocalizationFingerprint{
			Languages:       "en-AU",
			Locale:          "en-AU",
			AcceptLanguages: "en-AU,en;q=0.9",
		},
		Timezone: &TimezoneFingerprint{Zone: "Australia/Sydney"},
	},
	"NZ": {
		Localization: &LocalizationFingerprint{
			Languages:       "en-NZ",
			Locale:          "en-NZ",
			AcceptLanguages: "en-NZ,en;q=0.9",
		},
		Timezone: &TimezoneFingerprint{Zone: "Pacific/Auckland"},
	},

	// ── Western Europe ───────────────────────────────────────
	"GB": {
		Localization: &LocalizationFingerprint{
			Languages:       "en-GB",
			Locale:          "en-GB",
			AcceptLanguages: "en-GB,en;q=0.9,en-US;q=0.5",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/London"},
	},
	"IE": {
		Localization: &LocalizationFingerprint{
			Languages:       "en-IE",
			Locale:          "en-IE",
			AcceptLanguages: "en-IE,en;q=0.9,en-US;q=0.5",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Dublin"},
	},
	"DE": {
		Localization: &LocalizationFingerprint{
			Languages:       "de-DE",
			Locale:          "de-DE",
			AcceptLanguages: "de-DE,de;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Berlin"},
	},
	"AT": {
		Localization: &LocalizationFingerprint{
			Languages:       "de-AT",
			Locale:          "de-AT",
			AcceptLanguages: "de-AT,de;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Vienna"},
	},
	"CH": {
		Localization: &LocalizationFingerprint{
			Languages:       "de-CH",
			Locale:          "de-CH",
			AcceptLanguages: "de-CH,de;q=0.9,fr-CH;q=0.7,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Zurich"},
	},
	"FR": {
		Localization: &LocalizationFingerprint{
			Languages:       "fr-FR",
			Locale:          "fr-FR",
			AcceptLanguages: "fr-FR,fr;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Paris"},
	},
	"NL": {
		Localization: &LocalizationFingerprint{
			Languages:       "nl-NL",
			Locale:          "nl-NL",
			AcceptLanguages: "nl-NL,nl;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Amsterdam"},
	},
	"BE": {
		Localization: &LocalizationFingerprint{
			Languages:       "nl-BE",
			Locale:          "nl-BE",
			AcceptLanguages: "nl-BE,nl;q=0.9,fr-BE;q=0.7,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Brussels"},
	},
	"ES": {
		Localization: &LocalizationFingerprint{
			Languages:       "es-ES",
			Locale:          "es-ES",
			AcceptLanguages: "es-ES,es;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Madrid"},
	},
	"PT": {
		Localization: &LocalizationFingerprint{
			Languages:       "pt-PT",
			Locale:          "pt-PT",
			AcceptLanguages: "pt-PT,pt;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Lisbon"},
	},
	"IT": {
		Localization: &LocalizationFingerprint{
			Languages:       "it-IT",
			Locale:          "it-IT",
			AcceptLanguages: "it-IT,it;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Rome"},
	},

	// ── Northern Europe ──────────────────────────────────────
	"SE": {
		Localization: &LocalizationFingerprint{
			Languages:       "sv-SE",
			Locale:          "sv-SE",
			AcceptLanguages: "sv-SE,sv;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Stockholm"},
	},
	"NO": {
		Localization: &LocalizationFingerprint{
			Languages:       "nb-NO",
			Locale:          "nb-NO",
			AcceptLanguages: "nb-NO,nb;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Oslo"},
	},
	"DK": {
		Localization: &LocalizationFingerprint{
			Languages:       "da-DK",
			Locale:          "da-DK",
			AcceptLanguages: "da-DK,da;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Copenhagen"},
	},
	"FI": {
		Localization: &LocalizationFingerprint{
			Languages:       "fi-FI",
			Locale:          "fi-FI",
			AcceptLanguages: "fi-FI,fi;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Helsinki"},
	},

	// ── Central & Eastern Europe ─────────────────────────────
	"PL": {
		Localization: &LocalizationFingerprint{
			Languages:       "pl-PL",
			Locale:          "pl-PL",
			AcceptLanguages: "pl-PL,pl;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Warsaw"},
	},
	"CZ": {
		Localization: &LocalizationFingerprint{
			Languages:       "cs-CZ",
			Locale:          "cs-CZ",
			AcceptLanguages: "cs-CZ,cs;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Prague"},
	},
	"RO": {
		Localization: &LocalizationFingerprint{
			Languages:       "ro-RO",
			Locale:          "ro-RO",
			AcceptLanguages: "ro-RO,ro;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Bucharest"},
	},
	"BG": {
		Localization: &LocalizationFingerprint{
			Languages:       "bg-BG",
			Locale:          "bg-BG",
			AcceptLanguages: "bg-BG,bg;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Sofia"},
	},
	"HR": {
		Localization: &LocalizationFingerprint{
			Languages:       "hr-HR",
			Locale:          "hr-HR",
			AcceptLanguages: "hr-HR,hr;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Zagreb"},
	},
	"HU": {
		Localization: &LocalizationFingerprint{
			Languages:       "hu-HU",
			Locale:          "hu-HU",
			AcceptLanguages: "hu-HU,hu;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Budapest"},
	},
	"SK": {
		Localization: &LocalizationFingerprint{
			Languages:       "sk-SK",
			Locale:          "sk-SK",
			AcceptLanguages: "sk-SK,sk;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Bratislava"},
	},
	"RU": {
		Localization: &LocalizationFingerprint{
			Languages:       "ru-RU",
			Locale:          "ru-RU",
			AcceptLanguages: "ru-RU,ru;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Moscow"},
	},
	"UA": {
		Localization: &LocalizationFingerprint{
			Languages:       "uk-UA",
			Locale:          "uk-UA",
			AcceptLanguages: "uk-UA,uk;q=0.9,ru;q=0.7,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Kyiv"},
	},

	// ── East Asia ────────────────────────────────────────────
	"JP": {
		Localization: &LocalizationFingerprint{
			Languages:       "ja-JP",
			Locale:          "ja-JP",
			AcceptLanguages: "ja-JP,ja;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Asia/Tokyo"},
	},
	"KR": {
		Localization: &LocalizationFingerprint{
			Languages:       "ko-KR",
			Locale:          "ko-KR",
			AcceptLanguages: "ko-KR,ko;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Asia/Seoul"},
	},
	"CN": {
		Localization: &LocalizationFingerprint{
			Languages:       "zh-CN",
			Locale:          "zh-CN",
			AcceptLanguages: "zh-CN,zh;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Asia/Shanghai"},
	},
	"TW": {
		Localization: &LocalizationFingerprint{
			Languages:       "zh-TW",
			Locale:          "zh-TW",
			AcceptLanguages: "zh-TW,zh;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Asia/Taipei"},
	},

	// ── Southeast Asia ───────────────────────────────────────
	"TH": {
		Localization: &LocalizationFingerprint{
			Languages:       "th-TH",
			Locale:          "th-TH",
			AcceptLanguages: "th-TH,th;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Asia/Bangkok"},
	},
	"VN": {
		Localization: &LocalizationFingerprint{
			Languages:       "vi-VN",
			Locale:          "vi-VN",
			AcceptLanguages: "vi-VN,vi;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Asia/Ho_Chi_Minh"},
	},
	"ID": {
		Localization: &LocalizationFingerprint{
			Languages:       "id-ID",
			Locale:          "id-ID",
			AcceptLanguages: "id-ID,id;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Asia/Jakarta"},
	},
	"PH": {
		Localization: &LocalizationFingerprint{
			Languages:       "fil-PH",
			Locale:          "fil-PH",
			AcceptLanguages: "fil-PH,fil;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Asia/Manila"},
	},
	"MY": {
		Localization: &LocalizationFingerprint{
			Languages:       "ms-MY",
			Locale:          "ms-MY",
			AcceptLanguages: "ms-MY,ms;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Asia/Kuala_Lumpur"},
	},
	"SG": {
		Localization: &LocalizationFingerprint{
			Languages:       "en-SG",
			Locale:          "en-SG",
			AcceptLanguages: "en-SG,en;q=0.9,zh-SG;q=0.7,zh;q=0.5",
		},
		Timezone: &TimezoneFingerprint{Zone: "Asia/Singapore"},
	},

	// ── South Asia & Middle East ─────────────────────────────
	"IN": {
		Localization: &LocalizationFingerprint{
			Languages:       "hi-IN",
			Locale:          "hi-IN",
			AcceptLanguages: "hi-IN,hi;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Asia/Kolkata"},
	},
	"TR": {
		Localization: &LocalizationFingerprint{
			Languages:       "tr-TR",
			Locale:          "tr-TR",
			AcceptLanguages: "tr-TR,tr;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Europe/Istanbul"},
	},
	"SA": {
		Localization: &LocalizationFingerprint{
			Languages:       "ar-SA",
			Locale:          "ar-SA",
			AcceptLanguages: "ar-SA,ar;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Asia/Riyadh"},
	},
	"AE": {
		Localization: &LocalizationFingerprint{
			Languages:       "ar-AE",
			Locale:          "ar-AE",
			AcceptLanguages: "ar-AE,ar;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Asia/Dubai"},
	},
	"IL": {
		Localization: &LocalizationFingerprint{
			Languages:       "he-IL",
			Locale:          "he-IL",
			AcceptLanguages: "he-IL,he;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Asia/Jerusalem"},
	},

	// ── Africa ───────────────────────────────────────────────
	"ZA": {
		Localization: &LocalizationFingerprint{
			Languages:       "en-ZA",
			Locale:          "en-ZA",
			AcceptLanguages: "en-ZA,en;q=0.9,af;q=0.7,zu;q=0.5",
		},
		Timezone: &TimezoneFingerprint{Zone: "Africa/Johannesburg"},
	},
	"NG": {
		Localization: &LocalizationFingerprint{
			Languages:       "en-NG",
			Locale:          "en-NG",
			AcceptLanguages: "en-NG,en;q=0.9,ig;q=0.7,ha;q=0.5",
		},
		Timezone: &TimezoneFingerprint{Zone: "Africa/Lagos"},
	},
	"EG": {
		Localization: &LocalizationFingerprint{
			Languages:       "ar-EG",
			Locale:          "ar-EG",
			AcceptLanguages: "ar-EG,ar;q=0.9,en-US;q=0.5,en;q=0.3",
		},
		Timezone: &TimezoneFingerprint{Zone: "Africa/Cairo"},
	},
}
