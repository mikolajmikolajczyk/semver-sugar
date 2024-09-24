package semver

func BumpSemverVersion(version string, increment string, format string) (string, error) {

	v, err := ParseVersion(version)
	if err != nil {
		return "", err
	}
	inc, err := ParseIncrement(increment)
	if err != nil {
		return "", err
	}
	return v.bump(inc).format(format), nil
}
