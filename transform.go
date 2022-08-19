package query

import "reflect"

// Transform identifies a transformation function used to transform results from a row to an application
// model structure. Source is the structure prepared from the database row and Destination is the
// transformed output.
type Transform[Source, Destination any] func(Source) Destination

// The Identity function is a [Transform] function that returns the original value. This function can be used as the
// transform when the caller wishes to receive the source value.
func Identity[Source any](src Source) Source { return src }

// The Auto function is a [Transform] function that attempts to automatically convert from the source type to the
// destination type based on similarities between types.
func Auto[Source, Destination any](src Source) Destination {
	var dst Destination
	transformAuto(reflect.ValueOf(src), reflect.ValueOf(&dst).Elem())
	return dst
}

// transformAuto performs the automatic transformation. sval is the reflect value of the source variable and dval
// is the reflect value of the destination variable. The destination value must be settable.
func transformAuto(sval reflect.Value, dval reflect.Value) {
	styp := sval.Type()
	dtyp := dval.Type()
	if sval.Kind() != reflect.Struct || dval.Kind() != reflect.Struct {
		panic("source and destination must be struct types")
	}

	for i := 0; i < styp.NumField(); i++ {
		sf := styp.Field(i)
		df, ok := dtyp.FieldByName(sf.Name)
		if !ok {
			continue
		}
		if df.Type.Kind() == reflect.Struct && sf.Type.Kind() == reflect.Struct {
			transformAuto(sval.Field(i), dval.FieldByIndex(df.Index))
			continue
		}
		if df.Type != sf.Type {
			continue
		}
		dval.FieldByIndex(df.Index).Set(sval.Field(i))
	}
}
