package scalers

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type LemonScaler struct {
	metricType v2.MetricTargetType
	logger     logr.Logger
	metadata   LemonMetadata
}

type LemonMetadata struct {
	depthTarget          int
	simulatedActualDepth int
	activationLimit      int
	triggerIndex         int
}

func NewLemonScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	var depthTarget, simulatedActualDepth, activationLimit int

	if depthStr, ok := config.TriggerMetadata["depthTarget"]; !ok {
		return nil, fmt.Errorf("no depthTarget given")
	} else {
		depthTarget, err = strconv.Atoi(depthStr)
		if err != nil {
			return nil, errors.New("invalid value for depthTarget. Must be an int")
		}
	}

	if simulatedActualDepthStr, ok := config.TriggerMetadata["simulatedActualDepth"]; !ok {
		return nil, fmt.Errorf("no simulatedActualDepth given")
	} else {
		simulatedActualDepth, err = strconv.Atoi(simulatedActualDepthStr)
		if err != nil {
			return nil, errors.New("invalid value for simulatedActualDepth. Must be an int")
		}
	}

	if activationLimitStr, ok := config.TriggerMetadata["activationLimit"]; !ok {
		return nil, fmt.Errorf("no activationLimit given")
	} else {
		activationLimit, err = strconv.Atoi(activationLimitStr)
		if err != nil {
			return nil, errors.New("invalid value for activationLimit. Must be an int")
		}
	}

	return &LemonScaler{
		metricType: metricType,
		logger:     InitializeLogger(config, "lemon_scaler"),
		metadata:   LemonMetadata{depthTarget: depthTarget, activationLimit: activationLimit, simulatedActualDepth: simulatedActualDepth},
	}, nil
}

func (s *LemonScaler) Close(context.Context) error {
	return nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *LemonScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("lemon-%s", "scaaale"))),
		},
		Target: GetMetricTarget(s.metricType, int64(s.metadata.depthTarget)),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *LemonScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {

	metric := GenerateMetricInMili(metricName, float64(s.metadata.simulatedActualDepth))
	return []external_metrics.ExternalMetricValue{metric}, s.metadata.simulatedActualDepth > s.metadata.activationLimit, nil
}
