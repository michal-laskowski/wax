# WAX - GO to TS

Generate typescript typings from Golang types.

## Installation

go get github.com/michal-laskowski/extras/gots

## Usage

Just call gots.GenerateTypeDefinition.

See tests for details.

## Remarks

### Generics

For generics use tag `waxGeneric:""`.

Only one generic param is supported atm.

```golang
type TestGeneric[T any] struct {
 Data []T `waxGeneric:""`
}
```

will generate:

```typescript
type TestGeneric<T> {
    Data T[]
}
```
