package sources

import "strings"

func OpenAlexFilterPreset(name string) (map[string]string, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "systematic-review":
		return map[string]string{"filter": "type:article", "type": "article"}, true
	case "open-access-review":
		return map[string]string{"filter": "type:article,open_access.is_oa:true", "type": "article", "open-access": "true"}, true
	case "recent-domain-map":
		return map[string]string{"filter": "from_publication_date:2020-01-01,type:article", "from-year": "2020", "type": "article"}, true
	default:
		return nil, false
	}
}
