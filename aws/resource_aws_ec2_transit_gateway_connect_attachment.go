package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/keyvaluetags"
)

func resourceAwsEc2TransitGatewayConnect() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEc2TransitGatewayConnectCreate,
		Read:   resourceAwsEc2TransitGatewayConnectRead,
		Update: resourceAwsEc2TransitGatewayConnectUpdate,
		Delete: resourceAwsEc2TransitGatewayConnectDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"transit_gateway_attachment_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"transit_gateway_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"transit_gateway_owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"resource_owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"resource_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"association": {
				Type:     schema.TypeMap,
				Computed: true,
			},
			"creation_time": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsEc2TransitGatewayConnectCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	transitGatewayID := d.Get("transit_gateway_id").(string)

	input := &ec2.CreateTransitGatewayConnectInput{
		Options: &ec2.CreateTransitGatewayConnectRequestOptions{
			Protocol: aws.String(d.Get("protocol").(string)),
		},
		TransportTransitGatewayAttachmentId: aws.String(transitGatewayID),
		TagSpecifications:                   ec2TagSpecificationsFromMap(d.Get("tags").(map[string]interface{}), ec2.ResourceTypeTransitGatewayAttachment),
	}

	log.Printf("[DEBUG] Creating EC2 Transit Gateway Connect: %s", input)
	output, err := conn.CreateTransitGatewayConnect(input)
	if err != nil {
		return fmt.Errorf("error creating EC2 Transit Gateway Connect: %s", err)
	}

	d.SetId(aws.StringValue(output.TransitGatewayConnect.TransitGatewayAttachmentId))

	transitGateway, err := ec2DescribeTransitGateway(conn, transitGatewayID)
	if err != nil {
		return fmt.Errorf("error describing EC2 Transit Gateway (%s): %s", transitGatewayID, err)
	}

	if transitGateway.Options == nil {
		return fmt.Errorf("error describing EC2 Transit Gateway (%s): missing options", transitGatewayID)
	}

	return resourceAwsEc2TransitGatewayConnect(d, meta)
}

func resourceAwsEc2TransitGatewayConnectRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	transitGatewayConnect, err := ec2DescribeTransitGatewayConnect(conn, d.Id())

	if isAWSErr(err, "InvalidTransitGatewayConnect.NotFound", "") {
		log.Printf("[WARN] EC2 Transit Gateway Connect (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading EC2 Transit Gateway Connect: %s", err)
	}

	if transitGatewayConnect == nil {
		log.Printf("[WARN] EC2 Transit Gateway Connect (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	transitGatewayID := aws.StringValue(transitGatewayConnect.TransitGatewayId)
	transitGateway, err := ec2DescribeTransitGateway(conn, transitGatewayID)
	if err != nil {
		return fmt.Errorf("error describing EC2 Transit Gateway (%s): %s", transitGatewayID, err)
	}

	if transitGateway.Options == nil {
		return fmt.Errorf("error describing EC2 Transit Gateway (%s): missing options", transitGatewayID)
	}

	// // We cannot read Transit Gateway Route Tables for Resource Access Manager shared Transit Gateways
	// // Default these to a non-nil value so we can match the existing schema of Default: true
	// transitGatewayDefaultRouteTableAssociation := &ec2.TransitGatewayRouteTableAssociation{}
	// transitGatewayDefaultRouteTablePropagation := &ec2.TransitGatewayRouteTablePropagation{}
	// if aws.StringValue(transitGateway.OwnerId) == aws.StringValue(transitGatewayVpcAttachment.VpcOwnerId) {
	// 	transitGatewayAssociationDefaultRouteTableID := aws.StringValue(transitGateway.Options.AssociationDefaultRouteTableId)
	// 	transitGatewayDefaultRouteTableAssociation, err = ec2DescribeTransitGatewayRouteTableAssociation(conn, transitGatewayAssociationDefaultRouteTableID, d.Id())
	// 	if err != nil {
	// 		return fmt.Errorf("error determining EC2 Transit Gateway Attachment (%s) association to Route Table (%s): %s", d.Id(), transitGatewayAssociationDefaultRouteTableID, err)
	// 	}

	// 	transitGatewayPropagationDefaultRouteTableID := aws.StringValue(transitGateway.Options.PropagationDefaultRouteTableId)
	// 	transitGatewayDefaultRouteTablePropagation, err = ec2DescribeTransitGatewayRouteTablePropagation(conn, transitGatewayPropagationDefaultRouteTableID, d.Id())
	// 	if err != nil {
	// 		return fmt.Errorf("error determining EC2 Transit Gateway Attachment (%s) propagation to Route Table (%s): %s", d.Id(), transitGatewayPropagationDefaultRouteTableID, err)
	// 	}
	// }

	if transitGatewayConnect.Options == nil {
		return fmt.Errorf("error reading EC2 Transit Gateway Connect (%s): missing options", d.Id())
	}

	// d.Set("appliance_mode_support", transitGatewayVpcAttachment.Options.ApplianceModeSupport)
	// d.Set("dns_support", transitGatewayVpcAttachment.Options.DnsSupport)
	// d.Set("ipv6_support", transitGatewayVpcAttachment.Options.Ipv6Support)

	// if err := d.Set("subnet_ids", aws.StringValueSlice(transitGatewayVpcAttachment.SubnetIds)); err != nil {
	// 	return fmt.Errorf("error setting subnet_ids: %s", err)
	// }

	if err := d.Set("tags", keyvaluetags.Ec2KeyValueTags(transitGatewayConnect.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	// d.Set("transit_gateway_default_route_table_association", (transitGatewayDefaultRouteTableAssociation != nil))
	// d.Set("transit_gateway_default_route_table_propagation", (transitGatewayDefaultRouteTablePropagation != nil))
	d.Set("transit_gateway_id", aws.StringValue(transitGatewayConnect.TransitGatewayId))
	// d.Set("vpc_id", aws.StringValue(transitGatewayVpcAttachment.VpcId))
	// d.Set("vpc_owner_id", aws.StringValue(transitGatewayVpcAttachment.VpcOwnerId))

	return nil
}

func resourceAwsEc2TransitGatewayConnectUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// if d.HasChanges("appliance_mode_support", "dns_support", "ipv6_support", "subnet_ids") {
	// 	input := &ec2.ModifyTransitGatewayVpcAttachmentInput{
	// 		Options: &ec2.ModifyTransitGatewayVpcAttachmentRequestOptions{
	// 			ApplianceModeSupport: aws.String(d.Get("appliance_mode_support").(string)),
	// 			DnsSupport:           aws.String(d.Get("dns_support").(string)),
	// 			Ipv6Support:          aws.String(d.Get("ipv6_support").(string)),
	// 		},
	// 		TransitGatewayAttachmentId: aws.String(d.Id()),
	// 	}

	// 	oldRaw, newRaw := d.GetChange("subnet_ids")
	// 	oldSet := oldRaw.(*schema.Set)
	// 	newSet := newRaw.(*schema.Set)

	// 	if added := newSet.Difference(oldSet); added.Len() > 0 {
	// 		input.AddSubnetIds = expandStringSet(added)
	// 	}

	// 	if removed := oldSet.Difference(newSet); removed.Len() > 0 {
	// 		input.RemoveSubnetIds = expandStringSet(removed)
	// 	}

	// 	if _, err := conn.ModifyTransitGatewayVpcAttachment(input); err != nil {
	// 		return fmt.Errorf("error modifying EC2 Transit Gateway VPC Attachment (%s): %s", d.Id(), err)
	// 	}

	// 	if err := waitForEc2TransitGatewayVpcAttachmentUpdate(conn, d.Id()); err != nil {
	// 		return fmt.Errorf("error waiting for EC2 Transit Gateway VPC Attachment (%s) update: %s", d.Id(), err)
	// 	}
	// }

	// if d.HasChanges("transit_gateway_default_route_table_association", "transit_gateway_default_route_table_propagation") {
	// 	transitGatewayID := d.Get("transit_gateway_id").(string)

	// 	transitGateway, err := ec2DescribeTransitGateway(conn, transitGatewayID)
	// 	if err != nil {
	// 		return fmt.Errorf("error describing EC2 Transit Gateway (%s): %s", transitGatewayID, err)
	// 	}

	// 	if transitGateway.Options == nil {
	// 		return fmt.Errorf("error describing EC2 Transit Gateway (%s): missing options", transitGatewayID)
	// 	}

	// 	if d.HasChange("transit_gateway_default_route_table_association") {
	// 		if err := ec2TransitGatewayRouteTableAssociationUpdate(conn, aws.StringValue(transitGateway.Options.AssociationDefaultRouteTableId), d.Id(), d.Get("transit_gateway_default_route_table_association").(bool)); err != nil {
	// 			return fmt.Errorf("error updating EC2 Transit Gateway Attachment (%s) Route Table (%s) association: %s", d.Id(), aws.StringValue(transitGateway.Options.AssociationDefaultRouteTableId), err)
	// 		}
	// 	}

	// 	if d.HasChange("transit_gateway_default_route_table_propagation") {
	// 		if err := ec2TransitGatewayRouteTablePropagationUpdate(conn, aws.StringValue(transitGateway.Options.PropagationDefaultRouteTableId), d.Id(), d.Get("transit_gateway_default_route_table_propagation").(bool)); err != nil {
	// 			return fmt.Errorf("error updating EC2 Transit Gateway Attachment (%s) Route Table (%s) propagation: %s", d.Id(), aws.StringValue(transitGateway.Options.PropagationDefaultRouteTableId), err)
	// 		}
	// 	}
	// }

	if d.HasChange("tags") {
		o, n := d.GetChange("tags")

		if err := keyvaluetags.Ec2UpdateTags(conn, d.Id(), o, n); err != nil {
			return fmt.Errorf("error updating EC2 Transit Gateway Connect (%s) tags: %s", d.Id(), err)
		}
	}

	return nil
}

func resourceAwsEc2TransitGatewayConnectDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	input := &ec2.DeleteTransitGatewayConnectInput{
		TransitGatewayAttachmentId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting EC2 Transit Gateway Connect (%s): %s", d.Id(), input)
	_, err := conn.DeleteTransitGatewayConnect(input)

	if isAWSErr(err, "InvalidTransitGatewayAttachmentID.NotFound", "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting EC2 Transit Gateway Connect: %s", err)
	}

	return nil
}
