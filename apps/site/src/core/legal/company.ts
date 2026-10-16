// The operator of the site, as the offer, the footer and the bank need it.
// Fill in once the details are known; until `name` is set the footer shows
// no requisites and the documents show placeholders.
export interface Company {
  // Legal name: "ИП Иванов Иван Иванович" or "ОсОО «Ермектен»".
  name: string | null;
  // ИНН.
  taxId: string | null;
  // Registration number (ОКПО / state registration), if any.
  registrationNumber: string | null;
  address: string | null;
  email: string | null;
  phone: string | null;
}

export const COMPANY: Company = {
  name: null,
  taxId: null,
  registrationNumber: null,
  address: null,
  email: null,
  phone: null,
};
