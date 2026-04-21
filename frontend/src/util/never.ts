export function never(value: never): never {
  throw new Error(`Unexpected value: ${value}`);
}
