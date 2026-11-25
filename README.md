# okta-krm-extensions

Kubernetes kinds for the Okta configuration that Terraform's
`okta/okta` provider manages and Okta's operators do not. Okta publishes no Kubernetes operator for its configuration.

By the numbers: `okta/okta` is the #36 Terraform provider by downloads
(113,259,327); version 7.1.0 has 163 resources. Operators
cover 0. This repository adds **114 kinds** standing in for
**160** of them. Each kind is listed below with its Terraform equivalent;
the rest, and why, are under "Not covered".

**Limits.** Several Terraform types share one API resource: okta_app_oauth, okta_app_saml, okta_app_bookmark and the other app types are one Application kind whose signOnMode picks the type; likewise IdentityProvider (oidc, saml, social), Policy, PolicyRule and ClientRoleAssignment (the okta_oauth2_v1_clients_role_* types: the role type is the body's). A Terraform type that manages a set (okta_app_group_assignments, okta_group_owners, okta_app_signon_policy_rules, okta_request_conditions, okta_group_memberships, okta_user_group_memberships, okta_admin_role_targets, okta_policy_profile_enrollment_apps) is one object per member here, of the member's kind.

Some kinds manage less, or let go of more, than their Terraform type:

- `AppUserSchema` and `GroupSchema` are an app's user schema and the group schema, settings Okta changes with a partial POST: an object sets the properties it names under `definitions` (one, as a schema property type does, or several) and is compared on those alone. Deleting the object leaves its properties in the schema: Okta removes a custom property only when a POST sets it to null, which the runtime does not send, so remove a custom property in the Admin Console or the API (Terraform's custom property types remove it on destroy; its base property types, like these kinds, leave it).
- `AdminRoleGroupTarget` is a group target of a user's admin role (the `groups` of okta_admin_role_targets); app targets (its `apps`) are not covered yet.
- `AppPolicyAssignment` assigns an app to a policy (`PUT /api/v1/apps/{appId}/policies/{policyId}`), read from the policy's app list. Okta has no unassign: deleting the object leaves the app on the policy, as okta_app_access_policy_assignment does; okta_policy_profile_enrollment_apps instead moves the app back to the default profile enrollment policy, so make a second `AppPolicyAssignment` for that.
- `AppConnection` sets an app's provisioning connection; turning it on or off (`/lifecycle/activate`, `/lifecycle/deactivate`, okta_app_connection's `action`) is not managed.
- `AuthServerDefault`, `AuthServerClaimDefault`, `PolicyMfaDefault`, `PolicyPasswordDefault`, `PostAuthSessionPolicyRule` and `SessionViolationPolicyRule` take over what Okta made (the default authorization server, the claim named in `spec.name`, the system policy of the type, the rule named in `spec.name`): creating the object finds it and updates it, and deleting the object leaves it, as Terraform does.
- `ProfileMapping` is a profile mapping by its ID (`spec.mappingid`, from `GET /api/v1/mappings?sourceId=...&targetId=...`), where Terraform's okta_profile_mapping finds it by source and target; it updates the mapping's properties and, like the base property types, deleting the object leaves them. `UserBaseSchemaProperty` is the user schema (`spec.schemaid`: `default`), as `AppUserSchema` is an app's.
- `Feature`, `OrgFactor`, `OrgSupport`, `RoleSubscription` and `UserSubscription` are a state Okta sets by a call per value (`spec.status`, or `spec.support`: enable or disable, activate or deactivate, grant or revoke, subscribe or unsubscribe); deleting the object disables, deactivates, revokes or unsubscribes, as Terraform does. `OrgSupport` does not extend Okta Support's access (okta_org_support's `extend_by`). `Authenticator` takes over the authenticator Okta has for `spec.key`, updates it, activates or deactivates it as `spec.status` says, and deleting the object deactivates it; Terraform also creates one when none has the key, which this kind does not.
- `TrustedServer` is one server a custom authorization server trusts (Terraform's okta_trusted_server holds them all) and `IdentitySourceGroupMembership` one member of an identity source's group: one object each.
- `RequestV2` and `EndUserMyRequest` are access requests, which Okta cannot delete or change: deleting the object lets go of the request, as Terraform does.

**Readiness.** Every kind passes create, update, drift repair and delete
against a fake built from the same descriptors (`TestEveryKindOnFake`).
43 of them (readiness `live` below) passed the same against a real Okta
org too (`hack/live/records.json`); treat the others as untested against the
real API until they have.

## How it works

The kinds are generated, not hand-written. `hack/gen/vendor.yaml` pins
Okta's OpenAPI spec; [krm-rest-runtime](https://github.com/highfivelabs/krm-rest-runtime)'s
generator finds every resource the API exposes in a create/read/update/delete
shape, keeps those that stand in for a Terraform resource in the gap list
(`hack/gen/terraform-gaps.json`), and writes:

- `types.yaml`: per kind, its API calls, path parameters and Terraform equivalents
- `config/crds/`: one CRD per kind, its spec the API's request body
- `zz_generated.report.md`: the full list

One generic manager (`cmd/manager`) runs every kind on krm-rest-runtime's
reconciler: it creates the resource, reads it back every 10 minutes and
puts back anything changed outside Kubernetes, and deletes it with the
object. Spec fields keep Okta's own names, so the API reference
applies field for field. Path parameters are spec fields too.

Hand corrections to the automatic matching (a Terraform type matched to
the wrong API path, or not matched) are in `vendor.yaml`'s `overrides`.

Three specs feed the generator: Okta's management API (okta-sdk-golang),
Okta Identity Governance (`/governance/api/...`, okta-governance-sdk-golang,
the SDK the provider's governance resources call), and hand-written
supplements in `hack/gen/supplement/` for the few endpoints neither
describes, taken from the Terraform provider's own client code (each file
cites its sources).

## Install

```sh
kubectl kustomize config/default | KO_DOCKER_REPO=<registry> ko resolve -f - | kubectl apply --server-side -f -
```

Then, in each namespace that manages Okta, a Secret with
`apiToken` and a Connection named `default`
(`config/samples/connection.yaml`). Objects use it unless
`spec.connectionRef.name` names another. Secret values (tokens, passwords,
secret values) go in a Kubernetes Secret, not the spec:
`spec.secretRefs: {<field>: {name, key}}` fills that body field.

To run only some API groups, install those groups' CRDs from `config/crds`
and pass `--groups` to the manager.

**The Okta Admin domain.** `SecurityNotificationEmails` is served by the
org's admin host, `https://<org>-admin.okta.com`, not the org URL. Give it a
second Connection whose `baseURL` is the admin host (the same Secret and API
token work: Okta authorizes it with the same SSWS token) and name that
Connection in the object's `spec.connectionRef.name`.

**Okta Identity Governance** kinds (Campaign, EndUserMyRequest, Entitlement,
EntitlementBundle, Label, RequestCondition, RequestSettingOrganization,
RequestSettingResource, RequestV2) need an org licensed for it; the API token's admin
needs governance permissions.

## Kinds

| Kind | Terraform equivalent | Shape | Read | Readiness |
| --- | --- | --- | --- | --- |
| `admin.okta.hfl.dev/v1alpha1` `AdminRoleCustom` | `okta_admin_role_custom` | crud | `GET /api/v1/iam/roles/{roleIdOrLabel}` | live |
| `admin.okta.hfl.dev/v1alpha1` `AdminRoleCustomAssignments` | `okta_admin_role_custom_assignments` | crud | `GET /api/v1/iam/resource-sets/{resourceSetIdOrLabel}/bindings/{roleIdOrLabel}` | fake |
| `admin.okta.hfl.dev/v1alpha1` `AdminRoleGroupTarget` | `okta_admin_role_targets` | upsert | `GET /api/v1/users/{userId}/roles/{roleAssignmentId}/targets/groups` | fake |
| `agent.okta.hfl.dev/v1alpha1` `AgentPoolUpdate` | `okta_agent_pool_update` | crud | `GET /api/v1/agentPools/{poolId}/updates/{updateId}` | fake |
| `api.okta.hfl.dev/v1alpha1` `ApiServiceIntegration` | `okta_api_service_integration` | crud | `GET /integrations/api/v1/api-services/{apiServiceId}` | fake |
| `api.okta.hfl.dev/v1alpha1` `ApiToken` | `okta_api_token` | upsert | `GET /api/v1/api-tokens/{apiTokenId}` | fake |
| `app.okta.hfl.dev/v1alpha1` `AppConnection` | `okta_app_connection` | settings | `GET /api/v1/apps/{appId}/connections/default` | fake |
| `app.okta.hfl.dev/v1alpha1` `AppFeatures` | `okta_app_features` | settings | `GET /api/v1/apps/{appId}/features/{featureName}` | fake |
| `app.okta.hfl.dev/v1alpha1` `AppFederatedClaim` | `okta_app_federated_claim` | crud | `GET /api/v1/apps/{appId}/federated-claims/{claimId}` | live |
| `app.okta.hfl.dev/v1alpha1` `AppGroupAssignment` | `okta_app_group_assignment`, `okta_app_group_assignments` | upsert | `GET /api/v1/apps/{appId}/groups/{groupId}` | live |
| `app.okta.hfl.dev/v1alpha1` `AppOauthApiScope` | `okta_app_oauth_api_scope` | crud | `GET /api/v1/apps/{appId}/grants/{grantId}` | fake |
| `app.okta.hfl.dev/v1alpha1` `AppOauthPostLogoutRedirectUri` | `okta_app_oauth_post_logout_redirect_uri` | upsert | `GET /api/v1/apps/{appId}` | fake |
| `app.okta.hfl.dev/v1alpha1` `AppOauthRedirectUri` | `okta_app_oauth_redirect_uri` | upsert | `GET /api/v1/apps/{appId}` | fake |
| `app.okta.hfl.dev/v1alpha1` `AppPolicyAssignment` | `okta_app_access_policy_assignment`, `okta_policy_profile_enrollment_apps` | settings | `GET /api/v1/policies/{policyId}/app` | fake |
| `app.okta.hfl.dev/v1alpha1` `AppSamlAppSettings` | `okta_app_saml_app_settings` | settings | `GET /api/v1/apps/{appId}` | fake |
| `app.okta.hfl.dev/v1alpha1` `AppToken` | `okta_app_token` | settings | `GET /api/v1/apps/{appId}/tokens/{tokenId}` | fake |
| `app.okta.hfl.dev/v1alpha1` `AppUser` | `okta_app_user` | crud | `GET /api/v1/apps/{appId}/users/{userId}` | live |
| `app.okta.hfl.dev/v1alpha1` `AppUserSchema` | `okta_app_user_base_schema_property`, `okta_app_user_schema_property` | settings | `GET /api/v1/meta/schemas/apps/{appId}/default` | fake |
| `application.okta.hfl.dev/v1alpha1` `Application` | `okta_app_auto_login`, `okta_app_basic_auth`, `okta_app_bookmark`, `okta_app_oauth`, `okta_app_saml`, `okta_app_secure_password_store`, `okta_app_shared_credentials`, `okta_app_swa`, `okta_app_three_field` | crud | `GET /api/v1/apps/{appId}` | live |
| `auth.okta.hfl.dev/v1alpha1` `AuthServer` | `okta_auth_server` | crud | `GET /api/v1/authorizationServers/{authServerId}` | live |
| `auth.okta.hfl.dev/v1alpha1` `AuthServerClaim` | `okta_auth_server_claim` | crud | `GET /api/v1/authorizationServers/{authServerId}/claims/{claimId}` | live |
| `auth.okta.hfl.dev/v1alpha1` `AuthServerClaimDefault` | `okta_auth_server_claim_default` | crud | `GET /api/v1/authorizationServers/{authServerId}/claims/{claimId}` |
| `auth.okta.hfl.dev/v1alpha1` `AuthServerDefault` | `okta_auth_server_default` | crud | `GET /api/v1/authorizationServers/{authServerId}` |
| `auth.okta.hfl.dev/v1alpha1` `AuthServerPolicy` | `okta_auth_server_policy` | crud | `GET /api/v1/authorizationServers/{authServerId}/policies/{policyId}` | live |
| `auth.okta.hfl.dev/v1alpha1` `AuthServerPolicyRule` | `okta_auth_server_policy_rule` | crud | `GET /api/v1/authorizationServers/{authServerId}/policies/{policyId}/rules/{ruleId}` | live |
| `auth.okta.hfl.dev/v1alpha1` `AuthServerScope` | `okta_auth_server_scope` | crud | `GET /api/v1/authorizationServers/{authServerId}/scopes/{scopeId}` | live |
| `authenticator.okta.hfl.dev/v1alpha1` `Authenticator` | `okta_authenticator` | crud | `GET /api/v1/authenticators/{authenticatorId}` | fake |
| `authenticator.okta.hfl.dev/v1alpha1` `AuthenticatorMethodWebauthn` | `okta_authenticator_method_webauthn` | settings | `GET /api/v1/authenticators/{authenticatorId}/methods/{methodType}` | fake |
| `authenticator.okta.hfl.dev/v1alpha1` `AuthenticatorWebauthnCustomAaguid` | `okta_authenticator_webauthn_custom_aaguid` | crud | `GET /api/v1/authenticators/{authenticatorId}/aaguids/{aaguid}` | fake |
| `behavior.okta.hfl.dev/v1alpha1` `Behavior` | `okta_behavior` | crud | `GET /api/v1/behaviors/{behaviorId}` | live |
| `brand.okta.hfl.dev/v1alpha1` `Brand` | `okta_brand` | crud | `GET /api/v1/brands/{brandId}` | live |
| `campaign.okta.hfl.dev/v1alpha1` `Campaign` | `okta_campaign` | crud | `GET /governance/api/v1/campaigns/{campaignId}` | fake |
| `captcha.okta.hfl.dev/v1alpha1` `Captcha` | `okta_captcha` | crud | `GET /api/v1/captchas/{captchaId}` | live |
| `client.okta.hfl.dev/v1alpha1` `ClientRoleAssignment` | `okta_app_oauth_role_assignment`, `okta_oauth2_v1_clients_role_access_certifications_admin`, `okta_oauth2_v1_clients_role_access_requests_admin`, `okta_oauth2_v1_clients_role_api_access_management_admin`, `okta_oauth2_v1_clients_role_app_admin`, `okta_oauth2_v1_clients_role_custom`, `okta_oauth2_v1_clients_role_group_membership_admin`, `okta_oauth2_v1_clients_role_help_desk_admin`, `okta_oauth2_v1_clients_role_org_admin`, `okta_oauth2_v1_clients_role_read_only_admin`, `okta_oauth2_v1_clients_role_report_admin`, `okta_oauth2_v1_clients_role_super_admin`, `okta_oauth2_v1_clients_role_user_admin`, `okta_oauth2_v1_clients_role_workflows_admin` | crud | `GET /oauth2/v1/clients/{clientId}/roles/{roleAssignmentId}` | fake |
| `customized.okta.hfl.dev/v1alpha1` `CustomizedSigninPage` | `okta_customized_signin_page` | upsert | `GET /api/v1/brands/{brandId}/pages/sign-in/customized` | live |
| `device.okta.hfl.dev/v1alpha1` `Device` | `okta_device` | settings | `GET /api/v1/devices/{deviceId}` | fake |
| `domain.okta.hfl.dev/v1alpha1` `Domain` | `okta_domain` | crud | `GET /api/v1/domains/{domainId}` | live |
| `domain.okta.hfl.dev/v1alpha1` `DomainCertificate` | `okta_domain_certificate` | settings | `GET /api/v1/domains/{domainId}` | fake |
| `domain.okta.hfl.dev/v1alpha1` `DomainVerification` | `okta_domain_verification` | settings | `GET /api/v1/domains/{domainId}` | fake |
| `email.okta.hfl.dev/v1alpha1` `EmailCustomization` | `okta_email_customization` | crud | `GET /api/v1/brands/{brandId}/templates/email/{templateName}/customizations/{customizationId}` | fake |
| `email.okta.hfl.dev/v1alpha1` `EmailDomain` | `okta_email_domain` | crud | `GET /api/v1/email-domains/{emailDomainId}` | live |
| `email.okta.hfl.dev/v1alpha1` `EmailDomainVerification` | `okta_email_domain_verification` | settings | `GET /api/v1/email-domains/{emailDomainId}` | fake |
| `email.okta.hfl.dev/v1alpha1` `EmailSmtpServer` | `okta_email_smtp_server` | crud | `GET /api/v1/email-servers/{emailServerId}` | live |
| `email.okta.hfl.dev/v1alpha1` `EmailTemplateSettings` | `okta_email_template_settings` | settings | `GET /api/v1/brands/{brandId}/templates/email/{templateName}/settings` | fake |
| `end.okta.hfl.dev/v1alpha1` `EndUserMyRequest` | `okta_end_user_my_requests` | crud | `GET /governance/api/v2/my/catalogs/default/entries/{entryId}/requests/{requestId}` | fake |
| `entitlement.okta.hfl.dev/v1alpha1` `Entitlement` | `okta_entitlement` | crud | `GET /governance/api/v1/entitlements/{entitlementId}` | fake |
| `entitlement.okta.hfl.dev/v1alpha1` `EntitlementBundle` | `okta_entitlement_bundle` | crud | `GET /governance/api/v1/entitlement-bundles/{entitlementBundleId}` | fake |
| `event.okta.hfl.dev/v1alpha1` `EventHook` | `okta_event_hook` | crud | `GET /api/v1/eventHooks/{eventHookId}` | live |
| `event.okta.hfl.dev/v1alpha1` `EventHookVerification` | `okta_event_hook_verification` | settings | `GET /api/v1/eventHooks/{eventHookId}` | fake |
| `factor.okta.hfl.dev/v1alpha1` `Factor` | `okta_user_factor_question` | crud | `GET /api/v1/users/{userId}/factors/{factorId}` | live |
| `factor.okta.hfl.dev/v1alpha1` `FactorTotp` | `okta_factor_totp` | crud | `GET /api/v1/org/factors/hotp/profiles/{profileId}` | fake |
| `feature.okta.hfl.dev/v1alpha1` `Feature` | `okta_feature` | settings | `GET /api/v1/features/{featureId}` | fake |
| `group.okta.hfl.dev/v1alpha1` `Group` | `okta_group` | crud | `GET /api/v1/groups/{groupId}` | live |
| `group.okta.hfl.dev/v1alpha1` `GroupMembership` | `okta_group_memberships`, `okta_user_group_memberships` | upsert | `GET /api/v1/groups/{groupId}/users` | fake |
| `group.okta.hfl.dev/v1alpha1` `GroupOwner` | `okta_group_owner`, `okta_group_owners` | crud | `GET /api/v1/groups/{groupId}/owners` | fake |
| `group.okta.hfl.dev/v1alpha1` `GroupRole` | `okta_group_role` | crud | `GET /api/v1/groups/{groupId}/roles/{roleAssignmentId}` | live |
| `group.okta.hfl.dev/v1alpha1` `GroupRule` | `okta_group_rule` | crud | `GET /api/v1/groups/rules/{groupRuleId}` | live |
| `group.okta.hfl.dev/v1alpha1` `GroupSchema` | `okta_group_schema_property` | settings | `GET /api/v1/meta/schemas/group/default` | fake |
| `hook.okta.hfl.dev/v1alpha1` `HookKey` | `okta_hook_key` | crud | `GET /api/v1/hook-keys/{id}` | live |
| `identity.okta.hfl.dev/v1alpha1` `IdentityProvider` | `okta_idp_oidc`, `okta_idp_saml`, `okta_idp_social` | crud | `GET /api/v1/idps/{idpId}` | live |
| `identity.okta.hfl.dev/v1alpha1` `IdentitySourceGroup` | `okta_identity_source_group` | crud | `GET /api/v1/identity-sources/{identitySourceId}/groups/{groupOrExternalId}` | fake |
| `identity.okta.hfl.dev/v1alpha1` `IdentitySourceGroupMembership` | `okta_identity_source_group_membership` | crud | `GET /api/v1/identity-sources/{identitySourceId}/groups/{groupOrExternalId}/membership` | fake |
| `identity.okta.hfl.dev/v1alpha1` `IdentitySourceUser` | `okta_identity_source_user` | crud | `GET /api/v1/identity-sources/{identitySourceId}/users/{externalId}` | fake |
| `idp.okta.hfl.dev/v1alpha1` `IdpSamlKey` | `okta_idp_saml_key` | crud | `GET /api/v1/idps/credentials/keys/{kid}` | live |
| `inline.okta.hfl.dev/v1alpha1` `InlineHook` | `okta_inline_hook` | crud | `GET /api/v1/inlineHooks/{inlineHookId}` | live |
| `label.okta.hfl.dev/v1alpha1` `Label` | `okta_label` | crud | `GET /governance/api/v1/labels/{labelId}` | fake |
| `link.okta.hfl.dev/v1alpha1` `LinkDefinition` | `okta_link_definition` | crud | `GET /api/v1/meta/schemas/user/linkedObjects/{linkedObjectName}` | live |
| `link.okta.hfl.dev/v1alpha1` `LinkValue` | `okta_link_value` | set | `GET /api/v1/users/{primaryUserId}/linkedObjects/{associatedName}` | fake |
| `log.okta.hfl.dev/v1alpha1` `LogStream` | `okta_log_stream` | crud | `GET /api/v1/logStreams/{logStreamId}` | fake |
| `network.okta.hfl.dev/v1alpha1` `NetworkZone` | `okta_network_zone` | crud | `GET /api/v1/zones/{zoneId}` | live |
| `org.okta.hfl.dev/v1alpha1` `OrgCaptcha` | `okta_captcha_org_wide_settings`, `okta_org_captcha` | upsert | `GET /api/v1/org/captcha` | fake |
| `org.okta.hfl.dev/v1alpha1` `OrgConfiguration` | `okta_org_configuration` | settings | `GET /api/v1/org` | fake |
| `org.okta.hfl.dev/v1alpha1` `OrgFactor` | `okta_factor` | settings | `GET /api/v1/org/factors/{factorId}` | fake |
| `org.okta.hfl.dev/v1alpha1` `OrgSupport` | `okta_org_support` | settings | `GET /api/v1/org/privacy/oktaSupport` | fake |
| `policy.okta.hfl.dev/v1alpha1` `Policy` | `okta_app_signon_policy`, `okta_policy_mfa`, `okta_policy_password`, `okta_policy_profile_enrollment`, `okta_policy_signon` | crud | `GET /api/v1/policies/{policyId}` | live |
| `policy.okta.hfl.dev/v1alpha1` `PolicyDeviceAssurance` | `okta_policy_device_assurance_android`, `okta_policy_device_assurance_chromeos`, `okta_policy_device_assurance_ios`, `okta_policy_device_assurance_macos`, `okta_policy_device_assurance_windows` | crud | `GET /api/v1/device-assurances/{deviceAssuranceId}` | live |
| `policy.okta.hfl.dev/v1alpha1` `PolicyMfaDefault` | `okta_policy_mfa_default` | crud | `GET /api/v1/policies/{policyId}` |
| `policy.okta.hfl.dev/v1alpha1` `PolicyPasswordDefault` | `okta_policy_password_default` | crud | `GET /api/v1/policies/{policyId}` |
| `policy.okta.hfl.dev/v1alpha1` `PolicyRule` | `okta_app_signon_policy_rule`, `okta_app_signon_policy_rules`, `okta_entity_risk_policy_rule`, `okta_policy_rule_idp_discovery`, `okta_policy_rule_mfa`, `okta_policy_rule_password`, `okta_policy_rule_profile_enrollment`, `okta_policy_rule_signon` | crud | `GET /api/v1/policies/{policyId}/rules/{ruleId}` | live |
| `post.okta.hfl.dev/v1alpha1` `PostAuthSessionPolicyRule` | `okta_post_auth_session_policy_rule` | crud | `GET /api/v1/policies/{policyId}/rules/{ruleId}` |
| `preview.okta.hfl.dev/v1alpha1` `PreviewSigninPage` | `okta_preview_signin_page` | upsert | `GET /api/v1/brands/{brandId}/pages/sign-in/preview` | live |
| `principal.okta.hfl.dev/v1alpha1` `PrincipalRateLimit` | `okta_prinicipal_rate_limits` | settings | `GET /api/v1/principal-rate-limits/{principalRateLimitId}` | fake |
| `profile.okta.hfl.dev/v1alpha1` `ProfileMapping` | `okta_profile_mapping` | settings | `GET /api/v1/mappings/{mappingId}` |
| `push.okta.hfl.dev/v1alpha1` `PushGroup` | `okta_push_group` | crud | `GET /api/v1/apps/{appId}/group-push/mappings/{mappingId}` | fake |
| `push.okta.hfl.dev/v1alpha1` `PushProvider` | `okta_push_provider` | crud | `GET /api/v1/push-providers/{pushProviderId}` | live |
| `rate.okta.hfl.dev/v1alpha1` `RateLimitAdminNotificationSettings` | `okta_rate_limit_admin_notification_settings` | settings | `GET /api/v1/rate-limit-settings/admin-notifications` | live |
| `rate.okta.hfl.dev/v1alpha1` `RateLimitWarningThresholdPercentage` | `okta_rate_limit_warning_threshold_percentage` | settings | `GET /api/v1/rate-limit-settings/warning-threshold` | live |
| `rate.okta.hfl.dev/v1alpha1` `RateLimiting` | `okta_rate_limiting` | settings | `GET /api/v1/rate-limit-settings/per-client` | fake |
| `realm.okta.hfl.dev/v1alpha1` `Realm` | `okta_realm` | crud | `GET /api/v1/realms/{realmId}` | fake |
| `realm.okta.hfl.dev/v1alpha1` `RealmAssignment` | `okta_realm_assignment` | crud | `GET /api/v1/realm-assignments/{assignmentId}` | fake |
| `request.okta.hfl.dev/v1alpha1` `RequestCondition` | `okta_request_condition`, `okta_request_conditions` | crud | `GET /governance/api/v2/resources/{resourceId}/request-conditions/{requestConditionId}` | fake |
| `request.okta.hfl.dev/v1alpha1` `RequestSequence` | `okta_request_sequence` | settings | `GET /governance/api/v2/resources/{resourceId}/request-sequences/{sequenceId}` | fake |
| `request.okta.hfl.dev/v1alpha1` `RequestSettingOrganization` | `okta_request_setting_organization` | settings | `GET /governance/api/v2/request-settings` | fake |
| `request.okta.hfl.dev/v1alpha1` `RequestSettingResource` | `okta_request_setting_resource` | settings | `GET /governance/api/v2/resources/{resourceId}/request-settings` | fake |
| `request.okta.hfl.dev/v1alpha1` `RequestV2` | `okta_request_v2` | crud | `GET /governance/api/v2/requests/{requestId}` | fake |
| `resource.okta.hfl.dev/v1alpha1` `ResourceSet` | `okta_resource_set` | crud | `GET /api/v1/iam/resource-sets/{resourceSetIdOrLabel}` | live |
| `resource.okta.hfl.dev/v1alpha1` `ResourceOwner` | `okta_resource_owner` | settings | never read (`POST /governance/api/v1/resource-owners`) | fake |
| `review.okta.hfl.dev/v1alpha1` `Review` | `okta_review` | settings | `GET /governance/api/v1/reviews/{reviewId}` | fake |
| `role.okta.hfl.dev/v1alpha1` `RoleSubscription` | `okta_role_subscription` | settings | `GET /api/v1/roles/{roleRef}/subscriptions/{notificationType}` | fake |
| `security.okta.hfl.dev/v1alpha1` `SecurityEventsProvider` | `okta_security_events_provider` | crud | `GET /api/v1/security-events-providers/{securityEventProviderId}` | live |
| `security.okta.hfl.dev/v1alpha1` `SecurityNotificationEmails` | `okta_security_notification_emails` | settings | `GET /api/internal/org/settings/security-notification-settings` | fake |
| `session.okta.hfl.dev/v1alpha1` `SessionViolationPolicyRule` | `okta_session_violation_policy_rule` | crud | `GET /api/v1/policies/{policyId}/rules/{ruleId}` |
| `template.okta.hfl.dev/v1alpha1` `TemplateSms` | `okta_template_sms` | crud | `GET /api/v1/templates/sms/{templateId}` | fake |
| `theme.okta.hfl.dev/v1alpha1` `Theme` | `okta_theme` | settings | `GET /api/v1/brands/{brandId}/themes/{themeId}` | live |
| `threat.okta.hfl.dev/v1alpha1` `ThreatInsightConfiguration` | `okta_threat_insight_settings` | settings | `GET /api/v1/threats/configuration` | fake |
| `trusted.okta.hfl.dev/v1alpha1` `TrustedOrigin` | `okta_trusted_origin` | crud | `GET /api/v1/trustedOrigins/{trustedOriginId}` | live |
| `trusted.okta.hfl.dev/v1alpha1` `TrustedServer` | `okta_trusted_server` | upsert | `GET /api/v1/authorizationServers/{authServerId}/associatedServers` | fake |
| `ui.okta.hfl.dev/v1alpha1` `UiSchema` | `okta_ui_schema` | crud | `GET /api/v1/meta/uischemas/{id}` | live |
| `user.okta.hfl.dev/v1alpha1` `User` | `okta_user` | crud | `GET /api/v1/users/{id}` | live |
| `user.okta.hfl.dev/v1alpha1` `UserAdminRoles` | `okta_user_admin_roles` | crud | `GET /api/v1/users/{userId}/roles/{roleAssignmentId}` | live |
| `user.okta.hfl.dev/v1alpha1` `UserBaseSchemaProperty` | `okta_user_base_schema_property`, `okta_user_schema_property` | settings | `GET /api/v1/meta/schemas/user/{schemaId}` |
| `user.okta.hfl.dev/v1alpha1` `UserRisk` | `okta_user_risk` | settings | `GET /api/v1/users/{userId}/risk` | fake |
| `user.okta.hfl.dev/v1alpha1` `UserSubscription` | `okta_user_subscription` | settings | `GET /api/v1/users/{userId}/subscriptions/{notificationType}` | fake |
| `user.okta.hfl.dev/v1alpha1` `UserType` | `okta_user_type` | crud | `GET /api/v1/meta/types/user/{typeId}` | live |

## Not covered

<!-- BEGIN generated: not covered (krmgen.py, from vendor.yaml hardBlocks and draftNotes) -->
3 of the 163 Terraform gap types have no kind: 3 are not built (`hardBlocks` in `hack/gen/vendor.yaml`).

| Terraform | Status | Why |
| --- | --- | --- |
| `okta_email_sender` | not built | A retired API: terraform-provider-okta 7.1.0 marks it deprecated ("The api for this resource has been deprecated. Please use okta_email_domain instead"), and its calls (/api/v1/org/email/sender) are in neither of Okta's published OpenAPI specs, only the provider's private supplement. EmailDomain replaces it. |
| `okta_email_sender_verification` | not built | A retired API, as okta_email_sender's ("Please use okta_email_domain_verification instead"): EmailDomainVerification replaces it. |
| `okta_identity_source_import` | not built | Not a resource: an import session (create a session, upload the users and groups, start the import), which then ends; Okta keeps nothing to read back or delete. |
<!-- END generated: not covered -->

Some kinds act on a resource another kind or Okta makes, as Terraform's
types do:

- `DomainVerification`, `EmailDomainVerification`, `EventHookVerification`:
  the domain's, email domain's or event hook's verification status, asked
  for in `spec`; the verify call is made again on each sync until the
  status is that. Deleting the object leaves the resource as it is.
- `AppOauthRedirectUri`, `AppOauthPostLogoutRedirectUri`: one URI in an
  OIDC app's `settings.oauthClient` lists, written by a PUT of the whole
  app as just read (as Terraform does). Do not also set those lists on the
  `Application` kind.
- `AppSamlAppSettings`: a SAML app's `settings.app`, whole, written by a
  PUT of the whole app as just read, `settings.app` replaced (as Terraform
  does; Terraform's `settings` JSON string is the object `app`). Deleting
  the object leaves the app's settings as they are. Do not also set
  `settings.app` on the `Application` kind.
- `LinkValue`: every user associated with one primary user in a linked
  object relationship (each linked by its own PUT, unlinked by its own
  DELETE); the spec names the relationship's associated name too, which
  Terraform looks up in the link definition. A user linked elsewhere and
  not listed is unlinked.
- `Device`, `AppToken`, `RequestSequence`: made by Okta Verify, sign-ins
  and Okta Workflows, never through the API: the object names one by its
  IDs. A device's `status` is moved by its lifecycle calls (a suspended
  device is put back by DEACTIVATED, then ACTIVE); deleting an `AppToken`
  or `RequestSequence` object revokes or deletes it, as Terraform's destroy
  does, while deleting a `Device` object leaves the device.
- `DomainCertificate`: the certificate is never read back: it is sent
  again only when the spec changes.
- `Review`: reassigns a campaign's review to `reviewerId`; the reviewer is
  read back under another name, so a reassignment made elsewhere is not put
  back until the spec changes.
- `ResourceOwner`: sets the owners (`principalOrns`) of up to 10 governed
  resources (`resourceOrns`) with `POST /governance/api/v1/resource-owners`,
  which replaces theirs. It is never read, as Terraform never reads it:
  Okta lists owners only filtered by each resource's parent ORN, which
  neither the spec nor the write names. The POST is sent when the spec
  changes and only then, so owners changed outside Kubernetes are not seen
  or put back; `status.atProvider` is what the last POST answered.
  Deleting the object leaves the owners (Terraform's delete does nothing
  too); to remove them, set `principalOrns: []` first.

## For agents (MCP)

`cmd/mcp` serves this repository's kinds over the Model Context Protocol
(stdio), from `types.yaml`, `config/crds` and the live tests' records.
`.mcp.json` registers it, so an agent session opened in this repository
has it without setup. Elsewhere, run it from the repository root:

```sh
go run ./cmd/mcp
```

Its tools: `list_kinds`, `find_kinds_for_terraform`, `describe_kind` (every
field, the REST calls, the live record), `example` (the live-tested spec
where there is one), `validate_manifest` (the CRD schema, as the API server
checks it) and `plan_on_fake` (the controller run against the fake API,
with every REST call it makes). Nothing it does reaches the real API.
krm-rest-runtime's README ("For agents") has the details.

## Regenerate

```sh
hack/gen/generate.sh   # needs python3 + pyyaml, and krm-rest-runtime beside this repo
go test ./...
```
