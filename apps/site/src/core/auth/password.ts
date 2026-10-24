// bcrypt limits passwords by their encoded byte length, not character count.
export function passwordFitsBcrypt(password: string): boolean {
  return new TextEncoder().encode(password).length <= 72;
}
