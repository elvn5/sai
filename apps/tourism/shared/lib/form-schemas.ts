import { z } from 'zod';

// Number inputs bound with v-model.number yield '' when cleared, and native
// selects always yield strings; normalize both before validating.
export function optionalNumber(schema: z.ZodNumber): z.ZodOptional<z.ZodNumber> {
  const normalized = z.preprocess((value) => {
    if (value === '' || value === null || value === undefined) return undefined;
    const number = typeof value === 'string' ? Number(value) : value;
    return Number.isNaN(number) ? undefined : number;
  }, schema.optional());
  // preprocess() widens the input type to unknown, which would make the bound
  // inputs untyped; the runtime behavior is unchanged.
  return normalized as unknown as z.ZodOptional<z.ZodNumber>;
}

export function contactSchemas(messages: { phone: string; telegram: string }) {
  return {
    phone: z
      .string()
      .regex(/^\+?[0-9\s\-()]{5,32}$/, messages.phone)
      .optional()
      .or(z.literal('')),
    whatsapp: z
      .string()
      .regex(/^\+?[0-9\s\-()]{5,32}$/, messages.phone)
      .optional()
      .or(z.literal('')),
    telegram: z
      .string()
      .regex(/^@?[A-Za-z0-9_]{5,32}$/, messages.telegram)
      .optional()
      .or(z.literal('')),
  };
}

export function emptyToNull(value: string | undefined | null): string | null {
  const trimmed = value?.trim();
  return trimmed ? trimmed : null;
}
