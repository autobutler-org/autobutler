export const arrayCompare = <T>(
  arr1: T[] | undefined,
  arr2: T[] | undefined,
  elementComparator?: (a: T, b: T) => number,
): number => {
  elementComparator =
    elementComparator ??
    ((a, b) => {
      if (a < b) {
        return -1;
      }
      if (a > b) {
        return 1;
      }
      return 0;
    });
  if (arr1 === undefined && arr2 === undefined) {
    return 0;
  }
  if (arr1 === undefined) {
    return -1;
  }
  if (arr2 === undefined) {
    return 1;
  }

  const len1 = arr1.length;
  const len2 = arr2.length;
  const minLen = Math.min(len1, len2);

  for (let i = 0; i < minLen; i++) {
    const element1 = arr1[i] as T;
    const element2 = arr2[i] as T;
    const cmp = elementComparator(element1, element2);
    if (cmp !== 0) {
      return cmp;
    }
  }

  if (len1 < len2) {
    return -1;
  }
  if (len1 > len2) {
    return 1;
  }
  return 0;
};

export const arrayEquals = <T>(
  arr1: T[] | undefined,
  arr2: T[] | undefined,
  elementEquality?: (a: T, b: T) => boolean,
): boolean => {
  elementEquality = elementEquality ?? ((a, b) => a === b);
  if (arr1 === undefined && arr2 === undefined) {
    return true;
  }
  if (arr1 === undefined || arr2 === undefined) {
    return false;
  }
  if (arr1.length !== arr2.length) {
    return false;
  }

  for (let i = 0; i < arr1.length; i++) {
    const element1 = arr1[i] as T;
    const element2 = arr2[i] as T;
    if (!elementEquality(element1, element2)) {
      return false;
    }
  }

  return true;
};
