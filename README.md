# README #

A partial go implementation of Salesforce's [Lightning Platform REST API](https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/resources_list.htm) that provides create, update, delete, upsert and query commands to access/modify salesforce database tables.

Tables are modified using the SObject interface{}.  Structs representing table field data that implement the SObject interface maybe created using the genpkgs module.


### Example ###

[Create a Private Key and Self-Signed Digital Certificate](https://developer.salesforce.com/docs/atlas.en-us.sfdx_dev.meta/sfdx_dev/sfdx_dev_auth_key_and_cert.htm)


```
ctx := context.Background()
var contactID = "0141S0000009bv2QAA"
var contact salesforce.SObject = <your def package>.Contact{....}

sv, err := jwt.ServiceFromFile(ctx, cfg.SFCredentialFile, nil)
if err != nil {
    log.Fatalf("%v", err)
}
sv.Update(ctx, contact, contactID)

### Explain single operation, collection and batch ###

* create self-signed cert
* upload to SF
* sample config

### Create Object Definitions as struct ###

* demo gen config file

