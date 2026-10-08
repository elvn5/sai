// Popular Kyrgyzstan destinations, ordered roughly by tourist traffic.
// Stored as plain varchar (not a Postgres enum) so new places can be added
// without a migration.
export enum Destination {
  ISSYK_KUL = 'issyk_kul',
  KARAKOL = 'karakol',
  BISHKEK = 'bishkek',
  ALA_ARCHA = 'ala_archa',
  SONG_KOL = 'song_kol',
  NARYN = 'naryn',
  TASH_RABAT = 'tash_rabat',
  KEL_SUU = 'kel_suu',
  OSH = 'osh',
  ARSLANBOB = 'arslanbob',
  SARY_CHELEK = 'sary_chelek',
  TALAS = 'talas',
  BATKEN = 'batken',
  CHUY = 'chuy',
}

export enum CancellationPolicy {
  FREE = 'free',
  FLEXIBLE = 'flexible',
  STRICT = 'strict',
  NON_REFUNDABLE = 'non_refundable',
}

export enum ReviewTargetType {
  HOTEL = 'hotel',
  LOCAL_GUIDE = 'local_guide',
  TOUR = 'tour',
}
