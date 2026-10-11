package models

// Allowed values; apps/tourism/shared/lib/trail-options.ts mirrors them —
// keep the two in sync.
var (
	Areas = []string{
		"ala_archa", "alamedin", "chunkurchak", "issyk_ata", "kegeti", "sokuluk", "chon_kemin",
		"karakol", "jeti_oguz", "suusamyr", "song_kol", "kel_suu", "alay", "central_tien_shan", "other",
	}
	Kinds = []string{"hike", "trek", "climb", "ski_tour", "ice_climb", "rock_climb"}
	// Russian mountaineering grades; "trail" is a walk without a grade.
	Grades  = []string{"trail", "1a", "1b", "2a", "2b", "3a", "3b", "4a", "4b", "5a", "5b", "6a", "6b"}
	Seasons = []string{"spring", "summer", "autumn", "winter"}
	Gear    = []string{
		"helmet", "crampons", "microspikes", "ice_axe", "harness", "rope", "ice_screws",
		"trekking_poles", "headlamp", "tent", "sleeping_bag", "stove", "avalanche_kit",
		"snowshoes", "gaiters", "sunglasses",
	}

	TrailStates    = []string{"clear", "wet", "patchy_snow", "snow", "deep_snow", "ice"}
	Tractions      = []string{"none", "microspikes", "crampons"}
	AvalancheRisks = []string{"low", "moderate", "considerable", "high"}

	Transports   = []string{"car", "bus", "arranged"}
	Levels       = []string{"beginner", "intermediate", "advanced"}
	OutingStates = []string{"open", "cancelled"}

	TrailSorts = []string{"popular", "name", "elevation", "newest"}
)
