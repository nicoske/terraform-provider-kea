package dhcp4

import (
	stderrors "errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

// applied reports whether Kea applied the change, even when it could not
// write it to disk.
func applied(err error) bool {
	var writeErr *keadhcp4.ConfigWriteError
	return err == nil || stderrors.As(err, &writeErr)
}

// addConfigWriteError reports a change that Kea applied but could not write
// to disk. Called after the state is set, so that Terraform keeps the object
// and marks it tainted on create.
func addConfigWriteError(diags *diag.Diagnostics, err error) {
	if err != nil {
		diags.AddError("Error Writing Configuration", err.Error())
	}
}

// addDeleteError reports a failed delete. A delete that Kea applied but could
// not write to disk is a warning, so that Terraform still forgets the object.
func addDeleteError(diags *diag.Diagnostics, summary string, err error) {
	if applied(err) {
		diags.AddWarning("Error Writing Configuration", err.Error())
		return
	}
	diags.AddError(summary, err.Error())
}
