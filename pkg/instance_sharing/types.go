package instance_sharing

const (
	ReferencePlanDescription = "Allows to create a reference to a shared service instance from any environment in a subaccount and manage service bindings to that service instance."
	ReferencePlanName        = "reference-instance"

	ReferencedInstanceIDKey = "referenced_instance_id"

	SupportInstanceSharingKey = "supportInstanceSharing"

	ReferencePlanNameSelector     = "plan_name_selector"
	ReferenceInstanceNameSelector = "instance_name_selector"
	ReferenceLabelSelector        = "instance_labels_selector"
)
