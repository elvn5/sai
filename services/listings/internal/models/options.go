package models

// Allowed values for listing fields; apps/site/src/core/lib/listing-options.ts
// mirrors them — keep the two in sync.

var (
	Locales      = []string{"ru", "ky", "en", "de"}
	Destinations = []string{
		"issyk_kul", "karakol", "bishkek", "ala_archa", "song_kol", "naryn", "tash_rabat",
		"kel_suu", "osh", "arslanbob", "sary_chelek", "talas", "batken", "chuy",
	}
	HotelTypes           = []string{"hotel", "guesthouse", "hostel", "yurt_camp", "resort", "apartment"}
	CancellationPolicies = []string{"free", "flexible", "strict", "non_refundable"}
	HotelAmenities       = []string{
		"wifi", "breakfast", "parking", "airport_transfer", "hot_water", "heating", "kitchen",
		"restaurant", "sauna", "pool", "beach", "laundry", "family_friendly", "pets_allowed",
		"tours_desk", "horse_riding",
	}
	GuideLanguages = []string{
		"ru", "ky", "en", "de", "fr", "es", "it", "zh", "ja", "ko", "tr", "ar", "kk", "uz",
	}
	GuideSpecializations = []string{
		"trekking", "horse_riding", "mountaineering", "culture", "nomad_life", "city", "photo",
		"ski", "jeep", "food",
	}
	TargetTypes = []string{"hotel", "local_guide", "tour"}

	HotelSorts      = []string{"rating", "price_asc", "price_desc", "newest"}
	LocalGuideSorts = []string{"rating", "price_asc", "price_desc", "experience", "newest"}
)
