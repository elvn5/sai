import type { ObjectLiteral, SelectQueryBuilder } from 'typeorm';

/**
 * Restricts a query to rows whose TypeORM `simple-array` column (stored as
 * comma-joined text) contains every one of `values`. Wrapping both sides in
 * commas keeps "ru" from matching inside e.g. "rus".
 */
export function whereSimpleArrayContainsAll<T extends ObjectLiteral>(
  qb: SelectQueryBuilder<T>,
  column: string,
  values: readonly string[],
  paramPrefix: string,
): void {
  values.forEach((value, index) => {
    const param = `${paramPrefix}${index}`;
    qb.andWhere(`(',' || ${column} || ',') LIKE :${param}`, { [param]: `%,${value},%` });
  });
}
