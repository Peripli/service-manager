package instance_sharing

const (
	ReferencePlanDescription = "Allows to create a reference to a shared service instance from any environment in a subaccount and manage service bindings to that service instance."
	ReferencePlanName        = "reference-instance"

	SupportsInstanceSharingKey = "supportsInstanceSharing"

	ReferencedInstanceIDKey         = "referenced_instance_id"
	ReferencedInstanceIDTitle       = "Reference Instance ID"
	ReferencedInstanceIDDescription = "Specify the ID of the instance to which you want to refer."

	ReferenceInstanceNameSelectorKey         = "instance_name_selector"
	ReferenceInstanceNameSelectorTitle       = "Find by instance name"
	ReferenceInstanceNameSelectorDescription = "You can use various search criteria to find shared instances to which you want to create a reference."

	ReferencePlanNameSelectorKey         = "plan_name_selector"
	ReferencePlanNameSelectorTitle       = "Find by plan name"
	ReferencePlanNameSelectorDescription = "You can use various search criteria to find shared instances to which you want to create a reference."

	ReferenceLabelSelectorKey         = "instance_labels_selector"
	ReferenceLabelSelectorTitle       = "Find by instance label"
	ReferenceLabelSelectorDescription = "You can use labels query to find shared instances to which you want to create a reference. For example: \"origin\": [\"eu\"] returns an instance whose origin is in the EU"
)
