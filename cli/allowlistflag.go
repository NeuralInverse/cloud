package cli

import (
	"encoding/csv"
	"strings"

	"github.com/spf13/pflag"
	"golang.org/x/xerrors"

	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

var (
	_ pflag.SliceValue = &AllowListFlag{}
	_ pflag.Value      = &AllowListFlag{}
)

// AllowListFlag implements pflag.SliceValue for nicloudsdk.APIAllowListTarget entries.
type AllowListFlag []nicloudsdk.APIAllowListTarget

func AllowListFlagOf(al *[]nicloudsdk.APIAllowListTarget) *AllowListFlag {
	return (*AllowListFlag)(al)
}

func (a AllowListFlag) String() string {
	return strings.Join(a.GetSlice(), ",")
}

func (a AllowListFlag) Value() []nicloudsdk.APIAllowListTarget {
	return []nicloudsdk.APIAllowListTarget(a)
}

func (AllowListFlag) Type() string { return "allow-list" }

func (a *AllowListFlag) Set(set string) error {
	values, err := csv.NewReader(strings.NewReader(set)).Read()
	if err != nil {
		return xerrors.Errorf("parse allow list entries as csv: %w", err)
	}
	for _, v := range values {
		if err := a.Append(v); err != nil {
			return err
		}
	}
	return nil
}

func (a *AllowListFlag) Append(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return xerrors.New("allow list entry cannot be empty")
	}
	var target nicloudsdk.APIAllowListTarget
	if err := target.UnmarshalText([]byte(value)); err != nil {
		return err
	}

	*a = append(*a, target)
	return nil
}

func (a *AllowListFlag) Replace(items []string) error {
	*a = []nicloudsdk.APIAllowListTarget{}
	for _, item := range items {
		if err := a.Append(item); err != nil {
			return err
		}
	}
	return nil
}

func (a *AllowListFlag) GetSlice() []string {
	out := make([]string, len(*a))
	for i, entry := range *a {
		out[i] = entry.String()
	}
	return out
}
