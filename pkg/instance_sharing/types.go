package instance_sharing

const (
	ReferencePlanDescription = "Allows to create a reference to a shared service instance from any environment in a subaccount and manage service bindings to that service instance."
	ReferencePlanName        = "reference-instance"

	SupportsInstanceSharingKey = "supportsInstanceSharing"

	ReferencedInstanceIDKey         = "referenced_instance_id"
	ReferencedInstanceIDTitle       = "Reference Instance ID"
	ReferencedInstanceIDDescription = "Find a reference instance by the provided instance ID."

	ReferenceInstanceNameSelectorKey         = "instance_name_selector"
	ReferenceInstanceNameSelectorTitle       = "Instance Name"
	ReferenceInstanceNameSelectorDescription = "Find a reference instance by the provided instance name."

	ReferencePlanNameSelectorKey         = "plan_name_selector"
	ReferencePlanNameSelectorTitle       = "Plan Name"
	ReferencePlanNameSelectorDescription = "Find a reference instance by the provided plan name."

	ReferenceLabelSelectorKey         = "instance_labels_selector"
	ReferenceLabelSelectorTitle       = "Instance Label"
	ReferenceLabelSelectorDescription = "Find a reference instance by the provided label and its value. For example: \"origin\": [\"eu\"] returns an instance whose origin is in the EU"
)
