package resource

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	securityGroupType = "aws_security_group"
	iamRoleType       = "aws_iam_role"
	instanceType      = "aws_instance"
	vpc               = "aws_vpc"

	yml = Config{
		iamRoleType: {
			Ids: []*string{aws.String("^foo.*")},
		},
		securityGroupType: {},
		instanceType: {
			Tags: map[string]string{
				"foo": "bar",
				"bla": "blub",
			},
		},
		vpc: {
			Ids: []*string{aws.String("^foo.*")},
			Tags: map[string]string{
				"foo": "bar",
			},
		},
	}

	f = &Filter{
		cfg: yml,
	}
)

func TestFilter_Validate(t *testing.T) {
	require.NoError(t, f.Validate(mockAWSClient()))
}

func TestFilter_Validate_EmptyConfig(t *testing.T) {
	require.NoError(t, f.Validate(mockAWSClient()))
}

func TestFilter_Validate_NotSupportedResourceTypeInConfig(t *testing.T) {
	f := &Filter{
		cfg: Config{
			securityGroupType:    {},
			"not_supported_type": {},
		},
	}

	require.Error(t, f.Validate(mockAWSClient()))
}

func TestFilter_ResourceTypes(t *testing.T) {
	resTypes := f.ResourceTypes()

	require.Len(t, resTypes, len(yml))
	require.Contains(t, resTypes, securityGroupType)
	require.Contains(t, resTypes, iamRoleType)
	require.Contains(t, resTypes, instanceType)
}

func TestFilter_ResourceTypes_emptyConfig(t *testing.T) {
	f := &Filter{
		cfg: Config{},
	}

	resTypes := f.ResourceTypes()

	require.Len(t, resTypes, 0)
	require.Empty(t, resTypes)
}

func TestFilter_matchID(t *testing.T) {
	r := FilterableResource{Type: iamRoleType, ID: "foo-lala"}

	matchesID, err := f.matchID(r)

	require.True(t, matchesID)
	require.NoError(t, err)
}

func TestFilter_matchID_ResourceIDnotMatchingFilterCriteria(t *testing.T) {
	r := FilterableResource{Type: iamRoleType, ID: "lala-foo"}

	matchesID, err := f.matchID(r)

	require.False(t, matchesID)
	require.NoError(t, err)
}

func TestFilter_matchID_NoFilterCriteriaSetForIds(t *testing.T) {
	r := FilterableResource{Type: securityGroupType, ID: "matches-any-id"}

	_, err := f.matchID(r)

	require.Error(t, err)
}

func TestFilter_MatchTags(t *testing.T) {
	var testCases = []struct {
		actual   FilterableResource
		expected bool
	}{
		{
			actual:   FilterableResource{Type: instanceType, Tags: map[string]string{"foo": "bar"}},
			expected: true,
		},
		{
			actual:   FilterableResource{Type: instanceType, Tags: map[string]string{"bla": "blub"}},
			expected: true,
		},
		{
			actual:   FilterableResource{Type: instanceType, Tags: map[string]string{"foo": "baz"}},
			expected: false,
		},
		{
			actual:   FilterableResource{Type: instanceType, Tags: map[string]string{"blub": "bla"}},
			expected: false,
		},
	}

	for _, tc := range testCases {
		matchesTags, err := f.matchTags(tc.actual)
		require.Equal(t, matchesTags, tc.expected)
		require.NoError(t, err)

	}
}

func TestResourceMatchTags_NoFilterCriteriaSetForTags(t *testing.T) {
	_, err := f.matchTags(FilterableResource{Type: securityGroupType, Tags: map[string]string{"any": "tag"}})

	require.Error(t, err)
}

func TestFilter_Matches(t *testing.T) {
	var testCases = []struct {
		actual   FilterableResource
		expected bool
	}{
		// only tag filter criteria given
		{
			actual:   FilterableResource{instanceType, "foo-lala", map[string]string{"foo": "bar"}},
			expected: true,
		},
		{
			actual:   FilterableResource{instanceType, "some-id", map[string]string{"any": "tag"}},
			expected: false,
		},
		{
			actual:   FilterableResource{Type: instanceType, ID: "some-id"},
			expected: false,
		},
		// only filter ID criteria given
		{
			actual:   FilterableResource{iamRoleType, "foo-lala", map[string]string{"any": "tag"}},
			expected: true,
		},
		{
			actual:   FilterableResource{iamRoleType, "some-id", map[string]string{"foo": "bar"}},
			expected: false,
		},
		{
			actual:   FilterableResource{Type: iamRoleType, ID: "some-id"},
			expected: false,
		},
		// ID and tag filter criteria
		{
			actual:   FilterableResource{vpc, "foo-lala", map[string]string{"any": "tag"}},
			expected: true,
		},
		{
			actual:   FilterableResource{vpc, "some-id", map[string]string{"foo": "bar"}},
			expected: true,
		},
		{
			actual:   FilterableResource{vpc, "some-id", map[string]string{"any": "tag"}},
			expected: false,
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expected, f.Matches(tc.actual))
	}
}

func TestMatch_NoFilterCriteriaGiven(t *testing.T) {
	assert.True(t, f.Matches(FilterableResource{securityGroupType, "any-id", map[string]string{"any": "tag"}}))
}
