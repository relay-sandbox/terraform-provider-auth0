package auth0

import (
	"net/http"

	"github.com/auth0/go-auth0/management"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func newBranding() *schema.Resource {
	return &schema.Resource{
		Create: createBranding,
		Read:   readBranding,
		Update: updateBranding,
		Delete: deleteBranding,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"colors": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"primary": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"page_background": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"favicon_url": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"logo_url": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"font": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"url": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"universal_login": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"body": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func createBranding(d *schema.ResourceData, m interface{}) error {
	d.SetId(resource.UniqueId())
	return updateBranding(d, m)
}

func readBranding(d *schema.ResourceData, m interface{}) error {
	api := m.(*management.Management)
	branding, err := api.Branding.Read()
	if err != nil {
		if mErr, ok := err.(management.Error); ok {
			if mErr.Status() == http.StatusNotFound {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	result := multierror.Append(
		d.Set("favicon_url", branding.FaviconURL),
		d.Set("logo_url", branding.LogoURL),
	)
	if _, ok := d.GetOk("colors"); ok {
		result = multierror.Append(result, d.Set("colors", flattenBrandingColors(branding.Colors)))
	}
	if _, ok := d.GetOk("font"); ok {
		result = multierror.Append(result, d.Set("font", flattenBrandingFont(branding.Font)))
	}

	tenant, err := api.Tenant.Read()
	if err != nil {
		return err
	}

	if tenant.Flags.EnableCustomDomainInEmails != nil && *tenant.Flags.EnableCustomDomainInEmails {
		if err := setUniversalLogin(d, m); err != nil {
			d.SetId("")
			return err
		}
	}

	return result.ErrorOrNil()
}

func updateBranding(d *schema.ResourceData, m interface{}) error {
	api := m.(*management.Management)

	branding := buildBranding(d)
	if err := api.Branding.Update(branding); err != nil {
		return err
	}

	universalLogin := buildBrandingUniversalLogin(d)
	if universalLogin.GetBody() != "" {
		if err := api.Branding.SetUniversalLogin(universalLogin); err != nil {
			return err
		}
	}

	return readBranding(d, m)
}

func deleteBranding(d *schema.ResourceData, m interface{}) error {
	api := m.(*management.Management)
	tenant, err := api.Tenant.Read()
	if err != nil {
		return err
	}

	if tenant.Flags.EnableCustomDomainInEmails != nil && *tenant.Flags.EnableCustomDomainInEmails {
		if err = api.Branding.DeleteUniversalLogin(); err != nil {
			if mErr, ok := err.(management.Error); ok {
				if mErr.Status() == http.StatusNotFound {
					d.SetId("")
					return nil
				}
			}
		}
	}

	return err
}

func buildBranding(d *schema.ResourceData) *management.Branding {
	branding := &management.Branding{
		FaviconURL: String(d, "favicon_url"),
		LogoURL:    String(d, "logo_url"),
	}

	List(d, "colors").Elem(func(d ResourceData) {
		branding.Colors = &management.BrandingColors{
			PageBackground: String(d, "page_background"),
			Primary:        String(d, "primary"),
		}
	})

	List(d, "font").Elem(func(d ResourceData) {
		branding.Font = &management.BrandingFont{
			URL: String(d, "url"),
		}
	})

	return branding
}

func buildBrandingUniversalLogin(d *schema.ResourceData) *management.BrandingUniversalLogin {
	universalLogin := &management.BrandingUniversalLogin{}

	List(d, "universal_login").Elem(func(d ResourceData) {
		universalLogin.Body = String(d, "body")
	})

	return universalLogin
}

func setUniversalLogin(d *schema.ResourceData, m interface{}) error {
	api := m.(*management.Management)
	universalLogin, err := api.Branding.UniversalLogin()
	if err != nil {
		if mErr, ok := err.(management.Error); ok {
			if mErr.Status() == http.StatusNotFound {
				return nil
			}
		}
		return err
	}

	return d.Set("universal_login", flattenBrandingUniversalLogin(universalLogin))
}

func flattenBrandingColors(brandingColors *management.BrandingColors) []interface{} {
	if brandingColors == nil {
		return nil
	}
	return []interface{}{
		map[string]interface{}{
			"page_background": brandingColors.PageBackground,
			"primary":         brandingColors.Primary,
		},
	}
}

func flattenBrandingUniversalLogin(brandingUniversalLogin *management.BrandingUniversalLogin) []interface{} {
	if brandingUniversalLogin == nil {
		return nil
	}
	return []interface{}{
		map[string]interface{}{
			"body": brandingUniversalLogin.Body,
		},
	}
}

func flattenBrandingFont(brandingFont *management.BrandingFont) []interface{} {
	if brandingFont == nil {
		return nil
	}
	return []interface{}{
		map[string]interface{}{
			"url": brandingFont.URL,
		},
	}
}
