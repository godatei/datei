import { Resource, linkedSignal, ResourceSnapshot, resourceFromSnapshots } from '@angular/core';

/**
 * Creates a resource that retains the previous value when loading.
 */
export function withPreviousValue<T>(input: Resource<T>): Resource<T> {
  // Adapted from https://github.com/angular/angular/pull/66328#issue-3782221575

  const derived = linkedSignal<ResourceSnapshot<T>, ResourceSnapshot<T>>({
    source: input.snapshot,
    computation: (snap, previous) => {
      if (snap.status === 'loading' && previous?.value && previous?.value.status !== 'error') {
        // When the input resource enters loading state, we keep the value
        // from its previous state, if any, unless the previous state was an error
        return { status: 'loading', value: previous.value.value };
      }

      // Otherwise we simply forward the state of the input resource.
      return snap;
    },
  });

  return resourceFromSnapshots(derived);
}
