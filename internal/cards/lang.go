package cards

import "fmt"

type LangCode string
type LangName string

const (
	EN LangCode = "EN" // English
	FR LangCode = "FR" // French
	DE LangCode = "DE" // German
	IT LangCode = "IT" // Italian
	PT LangCode = "PT" // Portuguese
	SP LangCode = "SP" // Spanish

	JP LangCode = "JP" // Japanese
	AE LangCode = "AE" // Asian-English
	KR LangCode = "KR" // Korean
	SC LangCode = "SC" // Simplified Chinese
)

const (
	English    LangName = "English"
	French     LangName = "Français"
	German     LangName = "Deutsch"
	Italian    LangName = "Italiano"
	Portuguese LangName = "Português"
	Spanish    LangName = "Español"

	Japanese          LangName = "日本語"
	AsianEnglish      LangName = "English (Asia)"
	Korean            LangName = "한국어"
	SimplifiedChinese LangName = "简体中文"
)

func GetLangName(code LangCode) (LangName, error) {
	switch code {
	case EN:
		return English, nil
	case FR:
		return French, nil
	case DE:
		return German, nil
	case IT:
		return Italian, nil
	case PT:
		return Portuguese, nil
	case SP:
		return Spanish, nil
	case JP:
		return Japanese, nil
	case AE:
		return AsianEnglish, nil
	case KR:
		return Korean, nil
	case SC:
		return SimplifiedChinese, nil
	default:
		return "", fmt.Errorf("unsupported language code: %s", code)
	}
}