package errors

func is(source Error, target error) bool {
	if err, ok := target.(Error); ok {
		return source.Error() == err.Error()
	}
	return false
}

func as(source Error, target any) bool {
	if err, ok := target.(*Error); ok {
		if source.Error() == (*err).Error() {
			*err = source
			return true
		}
	}
	return false
}

// ========================================

func combine(result *[]error, errors ...error) (changed bool) {
	// assert result != nil
	for _, err := range errors {
		combineAdd(result, &changed, err)
	}
	return
}

func combineAdd(result *[]error, changed *bool, err error) {
	if err == nil {
		return
	}
	if multiple, ok := err.(multipleErrors); ok {
		for _, inner := range multiple {
			combineAdd(result, changed, inner)
		}
	} else {
		*result = append(*result, err)
		*changed = true
	}
}

func concat(result *[]error, errors ...error) {
	// assert result != nil
	for _, err := range errors {
		concatAdd(result, err)
	}
	return
}

func concatAdd(result *[]error, err error) {
	if err == nil {
		return
	}
	if multiple, ok := err.(multipleErrors); ok {
		for _, inner := range multiple {
			concatAdd(result, inner)
		}
	} else {
		*result = append(*result, err)
	}
}
