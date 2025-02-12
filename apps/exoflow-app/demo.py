# https://developer.hashicorp.com/terraform/cdktf/api-reference/python/constructs#app
from cdktf import App, TerraformStack, TerraformOutput
from cdktf_cdktf_provider_aws.provider import AwsProvider
from cdktf_cdktf_provider_aws.ec2_instance import Ec2Instance
from cdktf_cdktf_provider_azurerm.provider import AzurermProvider
from cdktf_cdktf_provider_azurerm.linux_virtual_machine import LinuxVirtualMachine

class MultiCloudStack(TerraformStack):
    def __init__(self, scope: Construct, id: str):
        super().__init__(scope, id)

        # AWS Provider Configuration
        AwsProvider(self, "AWS", region="us-east-1")

        # Azure Provider Configuration
        AzurermProvider(self, "Azure", features={})

        # AWS EC2 Instance
        aws_instance = Ec2Instance(self, "AWSInstance",
                                   ami="ami-12345678",
                                   instance_type="t2.micro",
                                   tags={"Name": "AWSInstance"})

        # Azure VM
        azure_vm = LinuxVirtualMachine(self, "AzureVM",
                                       name="AzureVM",
                                       location="East US",
                                       resource_group_name="myResourceGroup",
                                       size="Standard_B1s",
                                       admin_username="adminuser",
                                       network_interface_ids=[],
                                       os_disk={ "caching": "ReadWrite", "storage_account_type": "Standard_LRS" },
                                       source_image_reference={ "publisher": "Canonical", "offer": "UbuntuServer", "sku": "18.04-LTS", "version": "latest" })

        # Outputs
        TerraformOutput(self, "aws_instance_id", value=aws_instance.id)
        TerraformOutput(self, "azure_vm_id", value=azure_vm.id)

app = App()
MultiCloudStack(app, "multi-cloud-stack")
app.synth()
