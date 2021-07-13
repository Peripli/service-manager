package instance_sharing

const (
	ReferencePlanDescription = "Allows to create a reference to a shared service instance from any environment in a subaccount and manage service bindings to that service instance."
	ReferencePlanName        = "reference-instance"

	SupportsInstanceSharingKey = "supportsInstanceSharing"

	ReferencedInstanceIDKey         = "referenced_instance_id"
	ReferencedInstanceIDTitle       = "Reference Instance ID"
	ReferencedInstanceIDDescription = "Specify the ID of the instance to which you want to create a reference."

	BySelectorsKey         = "referenced_instance_id"
	BySelectorsTitle       = "Reference Instance ID"
	BySelectorsDescription = "Specify the ID of the instance to which you want to create a reference."

	ReferenceInstanceNameSelectorKey         = "instance_name_selector"
	ReferenceInstanceNameSelectorTitle       = "Find by Instance Name"
	ReferenceInstanceNameSelectorDescription = "You can use the instance name to find the shared instance to which you want to create a reference."

	ReferencePlanNameSelectorKey         = "plan_name_selector"
	ReferencePlanNameSelectorTitle       = "Find by Plan Name"
	ReferencePlanNameSelectorDescription = "You can use the plan name to find the shared instance to which you want to create a reference."

	ReferenceLabelSelectorKey         = "instance_labels_selector"
	ReferenceLabelSelectorTitle       = "instance_labels_selector"
	ReferenceLabelSelectorDescription = "instance_labels_selector"
)
