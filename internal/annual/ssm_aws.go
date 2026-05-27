package annual

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// AWSSSMClient is the subset of the SDK ssm.Client we use.
type AWSSSMClient interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
	PutParameter(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error)
}

// NewAWSSSMAdapter wraps an SDK client so it satisfies ssmAPI.
func NewAWSSSMAdapter(c AWSSSMClient) ssmAPI {
	return &awsSSMAdapter{c: c}
}

type awsSSMAdapter struct {
	c AWSSSMClient
}

func (a *awsSSMAdapter) GetParameter(ctx context.Context, name string) (string, error) {
	out, err := a.c.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(name)})
	if err != nil {
		var nf *ssmtypes.ParameterNotFound
		if errors.As(err, &nf) {
			return "", ErrParameterNotFound
		}
		return "", err
	}
	if out.Parameter == nil || out.Parameter.Value == nil {
		return "", nil
	}
	return *out.Parameter.Value, nil
}

func (a *awsSSMAdapter) PutParameter(ctx context.Context, name, value string) error {
	_, err := a.c.PutParameter(ctx, &ssm.PutParameterInput{
		Name:      aws.String(name),
		Value:     aws.String(value),
		Type:      ssmtypes.ParameterTypeString,
		Tier:      ssmtypes.ParameterTierStandard,
		Overwrite: aws.Bool(true),
	})
	return err
}
