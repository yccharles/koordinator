/*
Copyright 2022 The Koordinator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validation

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation/field"
	schedconfig "k8s.io/kubernetes/pkg/scheduler/apis/config"

	"github.com/koordinator-sh/koordinator/apis/extension"
	"github.com/koordinator-sh/koordinator/pkg/scheduler/apis/config"
)

// ValidateLoadAwareSchedulingArgs validates that LoadAwareSchedulingArgs are correct.
func ValidateLoadAwareSchedulingArgs(args *config.LoadAwareSchedulingArgs) error {
	var allErrs field.ErrorList

	if args.NodeMetricExpirationSeconds != nil && *args.NodeMetricExpirationSeconds <= 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("nodeMetricExpiredSeconds"), *args.NodeMetricExpirationSeconds, "nodeMetricExpiredSeconds should be a positive value"))
	}

	if err := validateResourceWeights(args.ResourceWeights); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("resourceWeights"), args.ResourceWeights, err.Error()))
	}
	if err := validateResourceThresholds(args.UsageThresholds); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("usageThresholds"), args.UsageThresholds, err.Error()))
	}
	if err := validateEstimatedScalingFactors(args.EstimatedScalingFactors); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("estimatedScalingFactors"), args.EstimatedScalingFactors, err.Error()))
	}

	for resourceName := range args.ResourceWeights {
		if _, ok := args.EstimatedScalingFactors[resourceName]; !ok {
			allErrs = append(allErrs, field.NotFound(field.NewPath("estimatedScalingFactors"), resourceName))
			break
		}
	}

	if err := validateAggregatedArgs(args.Aggregated, field.NewPath("aggregated")); err != nil {
		allErrs = append(allErrs, err...)
	}

	if len(allErrs) == 0 {
		return nil
	}
	return allErrs.ToAggregate()
}

func validateAggregatedArgs(
	aggregated *config.LoadAwareSchedulingAggregatedArgs,
	fldPath *field.Path,
) field.ErrorList {
	var allErrs field.ErrorList

	if aggregated == nil {
		return nil
	}

	if err := validateResourceThresholds(aggregated.UsageThresholds); err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("usageThresholds"), aggregated.UsageThresholds, err.Error()))
	}

	if aggregated.UsageAggregationType != "" {
		if err := validateAggregationType(aggregated.UsageAggregationType, fldPath.Child("usageAggregationType")); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	if aggregated.UsageAggregatedDuration.Duration < 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("usageAggregatedDuration"),
			aggregated.UsageAggregatedDuration, "duration must be >= 0"))
	}

	if aggregated.ScoreAggregationType != "" {
		if err := validateAggregationType(aggregated.ScoreAggregationType, fldPath.Child("scoreAggregationType")); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	if aggregated.ScoreAggregatedDuration.Duration < 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("scoreAggregatedDuration"),
			aggregated.ScoreAggregatedDuration, "duration must be >= 0"))
	}

	return allErrs
}

func validateAggregationType(aggType extension.AggregationType, fldPath *field.Path) *field.Error {
	validTypes := []string{
		string(extension.AVG),
		string(extension.P50), string(extension.P90),
		string(extension.P95), string(extension.P99),
	}

	for _, t := range validTypes {
		if string(aggType) == t {
			return nil
		}
	}
	return field.NotSupported(fldPath, aggType, validTypes)
}

func validateResourceWeights(resources map[corev1.ResourceName]int64) error {
	for resourceName, weight := range resources {
		if weight <= 0 {
			return fmt.Errorf("resource Weight of %v should be a positive value, got %v", resourceName, weight)
		}
		if weight > 100 {
			return fmt.Errorf("resource Weight of %v should be less than 100, got %v", resourceName, weight)
		}
	}
	return nil
}

func validateResourceThresholds(thresholds map[corev1.ResourceName]int64) error {
	for resourceName, thresholdPercent := range thresholds {
		if thresholdPercent < 0 {
			return fmt.Errorf("resource Threshold of %v should be a positive value, got %v", resourceName, thresholdPercent)
		}
		if thresholdPercent > 100 {
			return fmt.Errorf("resource Threshold of %v should be less than 100, got %v", resourceName, thresholdPercent)
		}
	}
	return nil
}

func validateEstimatedScalingFactors(scalingFactors map[corev1.ResourceName]int64) error {
	for resourceName, scalingFactor := range scalingFactors {
		if scalingFactor <= 0 {
			return fmt.Errorf("estimated resource ScalingFactor of %v should be a positive value, got %v", resourceName, scalingFactor)
		}
		if scalingFactor > 100 {
			return fmt.Errorf("estimated resource ScalingFactor of %v should be less than 100, got %v", resourceName, scalingFactor)
		}
	}
	return nil
}

func ValidateElasticQuotaArgs(elasticArgs *config.ElasticQuotaArgs) error {
	for resName, q := range elasticArgs.DefaultQuotaGroupMax {
		if q.Cmp(*resource.NewQuantity(0, resource.DecimalSI)) == -1 {
			return fmt.Errorf("elasticQuotaArgs error, defaultQuotaGroupMax should be a positive value, resourceName:%v, got %v",
				resName, q)
		}
	}

	for resName, q := range elasticArgs.SystemQuotaGroupMax {
		if q.Cmp(*resource.NewQuantity(0, resource.DecimalSI)) == -1 {
			return fmt.Errorf("elasticQuotaArgs error, systemQuotaGroupMax should be a positive value, resourceName:%v, got %v",
				resName, q)
		}
	}

	if elasticArgs.DelayEvictTime.Duration < 0 {
		return fmt.Errorf("elasticQuotaArgs error, DelayEvictTime should be a positive value")
	}

	if elasticArgs.RevokePodInterval.Duration < 0 {
		return fmt.Errorf("elasticQuotaArgs error, RevokePodCycle should be a positive value")
	}

	return nil
}

func ValidateCoschedulingArgs(coeSchedulingArgs *config.CoschedulingArgs) error {
	if coeSchedulingArgs.DefaultTimeout.Duration < 0 {
		return fmt.Errorf("coeSchedulingArgs DefaultTimeoutSeconds invalid")
	}
	if coeSchedulingArgs.ControllerWorkers < 1 {
		return fmt.Errorf("coeSchedulingArgs ControllerWorkers invalid")
	}
	return nil
}

func validateResources(resources []schedconfig.ResourceSpec, p *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	for i, resource := range resources {
		if resource.Weight <= 0 || resource.Weight > 100 {
			msg := fmt.Sprintf("resource weight of %v not in valid range (0, 100]", resource.Name)
			allErrs = append(allErrs, field.Invalid(p.Index(i).Child("weight"), resource.Weight, msg))
		}
	}
	return allErrs
}

func ValidateDeviceShareArgs(path *field.Path, args *config.DeviceShareArgs) error {
	var allErrs field.ErrorList
	if args.ScoringStrategy != nil {
		allErrs = append(allErrs, validateResources(args.ScoringStrategy.Resources, path.Child("resources"))...)
	}

	if len(allErrs) == 0 {
		return nil
	}
	return allErrs.ToAggregate()
}

func ValidateReservationArgs(path *field.Path, args *config.ReservationArgs) error {
	var allErrs field.ErrorList

	if args.MinCandidateNodesPercentage < 0 || args.MinCandidateNodesPercentage > 100 {
		allErrs = append(allErrs, field.Invalid(
			path.Child("MinCandidateNodesPercentage"),
			args.MinCandidateNodesPercentage,
			"must be in the range [0, 100]",
		))
	}

	if args.MinCandidateNodesAbsolute < 0 {
		allErrs = append(allErrs, field.Invalid(
			path.Child("MinCandidateNodesAbsolute"),
			args.MinCandidateNodesAbsolute,
			"must be non-negative",
		))
	}

	if args.GCDurationSeconds < 0 {
		allErrs = append(allErrs, field.Invalid(
			path.Child("GcDuration"),
			args.GCDurationSeconds,
			"must be non-negative",
		))
	}

	if len(allErrs) == 0 {
		return nil
	}
	return allErrs.ToAggregate()
}

func ValidateNodeNUMAResourceArgs(path *field.Path, args *config.NodeNUMAResourceArgs) error {
	var allErrs field.ErrorList
	if args.DefaultCPUBindPolicy != "" &&
		args.DefaultCPUBindPolicy != config.CPUBindPolicyFullPCPUs &&
		args.DefaultCPUBindPolicy != config.CPUBindPolicySpreadByPCPUs {
		allErrs = append(allErrs, field.Invalid(path.Child("defaultCPUBindPolicy"), args.DefaultCPUBindPolicy, "must specified CPU bind policy FullPCPUs or SpreadByPCPUs"))
	}

	if args.ScoringStrategy == nil {
		allErrs = append(allErrs, field.Required(path.Child("scoringStrategy"), "scoring strategy must be specified"))
	} else {
		allErrs = append(allErrs, validateResources(args.ScoringStrategy.Resources, path.Child("resources"))...)
	}

	if args.NUMAScoringStrategy == nil {
		allErrs = append(allErrs, field.Required(path.Child("numaScoringStrategy"), "NUMA scoring strategy must be specified"))
	} else {
		allErrs = append(allErrs, validateResources(args.NUMAScoringStrategy.Resources, path.Child("resources"))...)
	}

	if len(allErrs) == 0 {
		return nil
	}
	return allErrs.ToAggregate()
}
