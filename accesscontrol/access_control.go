package accesscontrol

// Aerospike access control reconciliation of access control.

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"

	as "github.com/aerospike/aerospike-client-go/v7"
)

// logger type alias.
type logger = logr.Logger

const (

	// Error marker for user not found errors.
	userNotFoundErr = "Invalid user"

	// Error marker for role not found errors.
	roleNotFoundErr = "Invalid role"
)

// privilegeStringToAerospikePrivilege converts privilegeString to an Aerospike privilege.
func privilegeStringToAerospikePrivilege(privilegeStrings []string) (
	[]as.Privilege, error,
) {
	aerospikePrivileges := make([]as.Privilege, 0, len(privilegeStrings))

	for _, privilege := range privilegeStrings {
		parts := strings.Split(privilege, ".")
		if _, ok := Privileges[parts[0]]; !ok {
			// First part of the privilege is not part of defined privileges.
			return nil, fmt.Errorf("invalid privilege %s", privilege)
		}

		privilegeCode := parts[0]
		namespaceName := ""
		setName := ""
		nParts := len(parts)

		switch nParts {
		case 2:
			namespaceName = parts[1]

		case 3:
			namespaceName = parts[1]
			setName = parts[2]
		}

		var code = as.Read //nolint:ineffassign // type is a private type in the pkg

		switch privilegeCode {
		case "read":
			code = as.Read

		case "write":
			code = as.Write

		case "read-write":
			code = as.ReadWrite

		case "read-write-udf":
			code = as.ReadWriteUDF

		case "data-admin":
			code = as.DataAdmin

		case "sys-admin":
			code = as.SysAdmin

		case "user-admin":
			code = as.UserAdmin

		case "truncate":
			code = as.Truncate

		case "sindex-admin":
			code = as.SIndexAdmin

		case "udf-admin":
			code = as.UDFAdmin

		default:
			return nil, fmt.Errorf("unknown privilege %s", privilegeCode)
		}

		aerospikePrivilege := as.Privilege{
			Code: code, Namespace: namespaceName, SetName: setName,
		}
		aerospikePrivileges = append(aerospikePrivileges, aerospikePrivilege)
	}

	return aerospikePrivileges, nil
}

// AerospikePrivilegeToPrivilegeString converts aerospikePrivilege to controller spec privilege string.
func AerospikePrivilegeToPrivilegeString(aerospikePrivileges []as.Privilege) (
	[]string, error,
) {
	privileges := make([]string, 0, len(aerospikePrivileges))

	for _, aerospikePrivilege := range aerospikePrivileges {
		var buffer bytes.Buffer

		switch aerospikePrivilege.Code {
		case as.Read:
			buffer.WriteString("read")

		case as.Write:
			buffer.WriteString("write")

		case as.ReadWrite:
			buffer.WriteString("read-write")

		case as.ReadWriteUDF:
			buffer.WriteString("read-write-udf")

		case as.DataAdmin:
			buffer.WriteString("data-admin")

		case as.SysAdmin:
			buffer.WriteString("sys-admin")

		case as.UserAdmin:
			buffer.WriteString("user-admin")

		case as.Truncate:
			buffer.WriteString("truncate")

		case as.SIndexAdmin:
			buffer.WriteString("sindex-admin")

		case as.UDFAdmin:
			buffer.WriteString("udf-admin")

		default:
			return nil, fmt.Errorf(
				"unknown privilege code %v", aerospikePrivilege.Code,
			)
		}

		if aerospikePrivilege.Namespace != "" {
			buffer.WriteString(".")
			buffer.WriteString(aerospikePrivilege.Namespace)

			if aerospikePrivilege.SetName != "" {
				buffer.WriteString(".")
				buffer.WriteString(aerospikePrivilege.SetName)
			}
		}

		privileges = append(privileges, buffer.String())
	}

	return privileges, nil
}

// AerospikeAccessControlReconcileCmd commands needed to Reconcile a single access control entry,
// for example a role or a user.
type AerospikeAccessControlReconcileCmd interface {
	// Execute executes the command. The implementation should be idempotent.
	Execute(
		client *as.Client, adminPolicy *as.AdminPolicy, logger logger,
	) error
}

// AerospikeRoleCreateUpdate creates or updates an Aerospike role.
type AerospikeRoleCreateUpdate struct {
	// The role's name.
	Name string

	// The privileges to set for the role. These privileges and only these privileges will be granted to the role
	// after this operation.
	Privileges []string

	// The whitelist to set for the role. These whitelist addresses and only these whitelist addresses will be
	// granted to the role after this operation.
	Whitelist []string

	// The readQuota specifies the read query rate that is permitted for the current role.
	ReadQuota uint32

	// The writeQuota specifies the write rate that is permitted for the current role.
	WriteQuota uint32
}

// Execute creates a new Aerospike role or updates an existing one.
func (roleCreate AerospikeRoleCreateUpdate) Execute(
	client *as.Client, adminPolicy *as.AdminPolicy, logger logger,
) error {
	role, err := client.QueryRole(adminPolicy, roleCreate.Name)
	isCreate := false

	if err != nil {
		if strings.Contains(err.Error(), roleNotFoundErr) {
			isCreate = true
		} else {
			// Failure to query for the role.
			return fmt.Errorf(
				"error querying role %s: %v", roleCreate.Name, err,
			)
		}
	}

	if isCreate {
		return roleCreate.CreateRole(client, adminPolicy, logger)
	}

	return roleCreate.UpdateRole(
		client, adminPolicy, role, logger,
	)
}

// CreateRole creates a new Aerospike role.
func (roleCreate AerospikeRoleCreateUpdate) CreateRole(
	client *as.Client, adminPolicy *as.AdminPolicy, logger logger,
) error {
	logger.Info("Creating role", "role name", roleCreate.Name)

	aerospikePrivileges, err := privilegeStringToAerospikePrivilege(roleCreate.Privileges)
	if err != nil {
		return fmt.Errorf("could not create role %s: %v", roleCreate.Name, err)
	}

	if err = client.CreateRole(
		adminPolicy, roleCreate.Name, aerospikePrivileges, roleCreate.Whitelist,
		roleCreate.ReadQuota, roleCreate.WriteQuota,
	); err != nil {
		return fmt.Errorf("could not create role %s: %v", roleCreate.Name, err)
	}

	logger.Info("Created role", "role name", roleCreate.Name)

	return nil
}

// UpdateRole updates an existing Aerospike role.
func (roleCreate AerospikeRoleCreateUpdate) UpdateRole(
	client *as.Client, adminPolicy *as.AdminPolicy, role *as.Role,
	logger logger,
) error {
	// Update the role.
	logger.Info("Updating role", "role name", roleCreate.Name)

	// Find the privileges to drop.
	currentPrivileges, err := AerospikePrivilegeToPrivilegeString(role.Privileges)
	if err != nil {
		return fmt.Errorf("could not update role %s: %v", roleCreate.Name, err)
	}

	desiredPrivileges := roleCreate.Privileges
	privilegesToRevoke := SliceSubtract(currentPrivileges, desiredPrivileges)
	privilegesToGrant := SliceSubtract(desiredPrivileges, currentPrivileges)

	if len(privilegesToRevoke) > 0 {
		aerospikePrivileges, err := privilegeStringToAerospikePrivilege(privilegesToRevoke)
		if err != nil {
			return fmt.Errorf(
				"could not update role %s: %v", roleCreate.Name, err,
			)
		}

		if err := client.RevokePrivileges(
			adminPolicy, roleCreate.Name, aerospikePrivileges,
		); err != nil {
			return fmt.Errorf(
				"error revoking privileges for role %s: %v", roleCreate.Name,
				err,
			)
		}

		logger.Info(
			"Revoked privileges for role", "role name", roleCreate.Name,
			"privileges", privilegesToRevoke,
		)
	}

	if len(privilegesToGrant) > 0 {
		aerospikePrivileges, err := privilegeStringToAerospikePrivilege(privilegesToGrant)
		if err != nil {
			return fmt.Errorf(
				"could not update role %s: %v", roleCreate.Name, err,
			)
		}

		if err := client.GrantPrivileges(
			adminPolicy, roleCreate.Name, aerospikePrivileges,
		); err != nil {
			return fmt.Errorf(
				"error granting privileges for role %s: %v", roleCreate.Name,
				err,
			)
		}

		logger.Info(
			"Granted privileges to role", "role name", roleCreate.Name,
			"privileges", privilegesToGrant,
		)
	}

	if !reflect.DeepEqual(role.Whitelist, roleCreate.Whitelist) {
		// Set whitelist.
		if err := client.SetWhitelist(
			adminPolicy, roleCreate.Name, roleCreate.Whitelist,
		); err != nil {
			return fmt.Errorf(
				"error setting whitelist for role %s: %v", roleCreate.Name, err,
			)
		}
	}

	logger.Info("Updated role", "role name", roleCreate.Name)

	return nil
}

// AerospikeUserCreateUpdate creates or updates an Aerospike user.
type AerospikeUserCreateUpdate struct {
	// The user's name.
	Name string

	// The password to set. Required for create. Optional for update.
	Password *string

	// The roles to set for the user. These roles and only these roles will be granted to the user after this operation.
	Roles []string
}

// Execute creates a new Aerospike user or updates an existing one.
func (userCreate AerospikeUserCreateUpdate) Execute(
	client *as.Client, adminPolicy *as.AdminPolicy, logger logger,
) error {
	user, err := client.QueryUser(adminPolicy, userCreate.Name)
	isCreate := false

	if err != nil {
		if strings.Contains(err.Error(), userNotFoundErr) {
			isCreate = true
		} else {
			// Failure to query for the user.
			return fmt.Errorf(
				"error querying user %s: %v", userCreate.Name, err,
			)
		}
	}

	if isCreate {
		return userCreate.CreateUser(client, adminPolicy, logger)
	}

	return userCreate.UpdateUser(
		client, adminPolicy, user, logger,
	)
}

// CreateUser creates a new Aerospike user.
func (userCreate AerospikeUserCreateUpdate) CreateUser(
	client *as.Client, adminPolicy *as.AdminPolicy, logger logger,
) error {
	logger.Info("Creating user", "username", userCreate.Name)

	if userCreate.Password == nil {
		return fmt.Errorf(
			"error creating user %s. Password not specified", userCreate.Name,
		)
	}

	if err := client.CreateUser(
		adminPolicy, userCreate.Name, *userCreate.Password, userCreate.Roles,
	); err != nil {
		return fmt.Errorf("could not create user %s: %v", userCreate.Name, err)
	}

	logger.Info("Created user", "username", userCreate.Name)

	return nil
}

// UpdateUser updates an existing Aerospike user.
func (userCreate AerospikeUserCreateUpdate) UpdateUser(
	client *as.Client, adminPolicy *as.AdminPolicy, user *as.UserRoles,
	logger logger,
) error {
	// Update the user.
	logger.Info("Updating user", "username", userCreate.Name)

	if userCreate.Password != nil {
		logger.Info("Updating password for user", "username", userCreate.Name)

		if err := client.ChangePassword(
			adminPolicy, userCreate.Name, *userCreate.Password,
		); err != nil {
			return fmt.Errorf(
				"error updating password for user %s: %v", userCreate.Name, err,
			)
		}

		logger.Info("Updated password for user", "username", userCreate.Name)
	}

	// Find the roles to grant and revoke.
	currentRoles := user.Roles
	desiredRoles := userCreate.Roles
	rolesToRevoke := SliceSubtract(currentRoles, desiredRoles)
	rolesToGrant := SliceSubtract(desiredRoles, currentRoles)

	if len(rolesToRevoke) > 0 {
		if err := client.RevokeRoles(adminPolicy, userCreate.Name, rolesToRevoke); err != nil {
			return fmt.Errorf(
				"error revoking roles for user %s: %v", userCreate.Name, err,
			)
		}

		logger.Info(
			"Revoked roles for user", "username", userCreate.Name, "roles",
			rolesToRevoke,
		)
	}

	if len(rolesToGrant) > 0 {
		if err := client.GrantRoles(adminPolicy, userCreate.Name, rolesToGrant); err != nil {
			return fmt.Errorf(
				"error granting roles for user %s: %v", userCreate.Name, err,
			)
		}

		logger.Info(
			"Granted roles to user", "username", userCreate.Name, "roles",
			rolesToGrant,
		)
	}

	logger.Info("Updated user", "username", userCreate.Name)

	return nil
}

// AerospikeUserDrop drops an Aerospike user.
type AerospikeUserDrop struct {
	// The user's name.
	Name string
}

// Execute implements dropping the user.
func (userDrop AerospikeUserDrop) Execute(
	client *as.Client, adminPolicy *as.AdminPolicy, logger logger,
) error {
	logger.Info("Dropping user", "username", userDrop.Name)

	if err := client.DropUser(adminPolicy, userDrop.Name); err != nil {
		if !strings.Contains(err.Error(), userNotFoundErr) {
			// Failure to drop for the user.
			return fmt.Errorf("error dropping user %s: %v", userDrop.Name, err)
		}
	}

	logger.Info("Dropped user", "username", userDrop.Name)

	return nil
}

// AerospikeRoleDrop drops an Aerospike role.
type AerospikeRoleDrop struct {
	// The role's name.
	Name string
}

// Execute implements dropping the role.
func (roleDrop AerospikeRoleDrop) Execute(
	client *as.Client, adminPolicy *as.AdminPolicy, logger logger,
) error {
	logger.Info("Dropping role", "role", roleDrop.Name)

	if err := client.DropRole(adminPolicy, roleDrop.Name); err != nil {
		if !strings.Contains(err.Error(), roleNotFoundErr) {
			// Failure to drop for the role.
			return fmt.Errorf("error dropping role %s: %v", roleDrop.Name, err)
		}
	}

	logger.Info("Dropped role", "role", roleDrop.Name)

	return nil
}

// SliceSubtract removes elements of slice2 from slice1 and returns the result.
func SliceSubtract(slice1, slice2 []string) []string {
	var result []string

	for _, s1 := range slice1 {
		found := false

		for _, toSubtract := range slice2 {
			if s1 == toSubtract {
				found = true
				break
			}
		}

		if !found {
			// s1 not found. Should be retained.
			result = append(result, s1)
		}
	}

	return result
}
