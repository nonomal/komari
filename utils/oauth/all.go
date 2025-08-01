package oauth

import (
	_ "github.com/komari-monitor/komari/utils/oauth/factory"
	_ "github.com/komari-monitor/komari/utils/oauth/generic"
	_ "github.com/komari-monitor/komari/utils/oauth/github"
)

func All() {
	//empty function to ensure all OIDC providers are registered
}
