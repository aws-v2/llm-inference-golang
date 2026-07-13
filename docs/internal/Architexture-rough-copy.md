### TL:DR - Architexture of the sagemaker platform 

This document describes the execution architexture of the sagemaker platform
Although inpired by AWS-Sagemaker , the goalis not to clone SageMaker feature-for-feature.
Instead, the goal is to build a *general purpose workflow engine* capable of executing  arbitrary scripts on specialized 
VMs while automatically managing dependencies, artifacts and storage



## The Philosophy
The platformis a bit opinionated (sort of like docker 1)
The platform does not caer what the user's script does.
A script may:
    - Download a youtube video
    - scrape websites 
    - monitor stoc prices
    - Train a neural network
    - Fine-tune an LLM
    - Render Blender scenes etc,

The platform doesnt attempt to understand the the script, Instead it provides the execution environment and a simple sdk through which scripts excahnge data with the platform 
The platform's only resposibility is: "Provision compute, execute scripts, collect artifacts , store artifacts,destroy compute, rinse and  repeat"

## High-Level Architexture
pipeline -> Nodes ->Scripts ->Specialized vms ->Physical host,

### Pipeline
A Pipeline is simply acollection of nodes connectedtogether with 
the names ofthe nodes a re *semantic only*,Nothing stops a user from Traning inside an ingest node
ordownloading/ingesting data in the deployment node, or running all steps in  a single script ,  the labels simply 
provide readability

### Nodes
A node represents a logical stage of work for the neat freaks
the node itself doesnot execute the code, instead it contains one or more scripts 
the node defines the vm specifications, 
we have predefined nodes like the 
    ingest (256mb ram, 1vcpu,5gb storage), 
    clean(1gb ram, 2vcpu,10gb storage),
    train(4gb ram,4vcpus,10gb storage) 
    evaluate(4gb ram,4vcpus,10gb storage) 
    deploy(4gb ram,4vcpus,10gb storage) 
for  each node the user can customize the resuorces,
these nodes act as templates for the vms,

### Scripts
Scripts are the actual unit of of execution,
Each script:
    - has its own source code 
    - has its own dependencies 
    - executes independently
    - publishes artifacts
    Example 1:
        yuotube_downloader.py
        youtube_downloader.requirements.txt
    Example 2:
        stocks.py
        stocks.requirements.txt

Each script owns its dependencies
Thiscompletely eliminates dependecy conflicts 
Heres the the reasoning forthat:
    suppose stocks.py requires requests==2.31  while the youtube_downloader.py requires requests==3.1.x ,installing both into one environment would create conflicts
    instead every script executes in its own isolated environment


### Specialized virtual machines 
The sgm control plane provisions a vm before a node begins execution 
it  does this by sending a nats message tothe ec2 service
on this subject <env-profile>.ec2.task.provision  a payload like this, 

type ProvisionRequestPayload struct{
    Profile strign `json:"profile," binding:"required"`
    Node Node `json:"node"`
    SessionID `json:"session_id" binding:"required"`

}
type Node struct{
    Name    string `json:"name" binding:"required"`
    Ram     int     `json:"" binding:"required"`
    Cpu     int     `json:"" binding:"required"`
    Storage int     `json:"" binding:"required"`
    
}


the profile key is used to determine the best host to deploy the provisioned vm in. ie
imagine this,  we have 12 idle hosts server(physical servers) thier work isto provide resources for vms,
now imagine we categorize these physical server in to 3 groups,(better names for categories will come later) 
group 1(red)=  all pysical servers labeled "red" we provision specialized sagemaker vms
group 2(green)= all physical server labeled "green" we provision specialixed rds vms
group 2(blue)= all physical servers labeled "green" we provision noraml ec2 vms
(More details on the vm loadbalancing strategies on this link)

now since we're sending the provision call to the ec2 from the sagemaker the profile will be "sgm"

in the host labeled "red" we have a base image that has which has the particulars required to runthe scripts ie,python docker python3 runtime images ie. python:3.12-slim, and in the rare cases we wont usethe containers  to run the workload we have  python3 installed(nobody uses python2 so...) directly in the vm,
the sagemake maker vms comes with a modified files to sync up with the sdk(checkout the sagemaker sdk here ) the modifiedfiles system lookslike this 
/workspace/jobid --> this is mainfolder forevery sgm operation, when executing the the scripts in the  containers,  we will mount this directliry to the dockercontainer liek this 
--volume /workspace/jobid:/workspace/ inside the the container
    /code --> stores the users code, ie. the python scripts the user uploaded for executions (stocks.py,stocks.requirements.txt,) if the user has multiple scripts in one node they are put here for execution,  
    
    /input --> for nodes that requires somesort of starter data lets forexample take the cleaning node, it cleaning some data, we a re assuming that data is in our s3 (its a bit opninionated,laughing),so when you wantto get the data from our s3 tothe /workspace/jobid/input you define it in your script using the sdk somethign like this 
    from cloudsdkimport Context

    ctx = Context()

    ctx.node_input("bucketname") or ctx.node_input(["list","of","filesname"]),  if you specify a bucket name the sdk will export allthe files fromthat bucekt inot the input folder, if you dinfeina file name it only exports said file 

    /models -->this folder has no much difference from the /ouput folder as it holds the ouput of the train/evaluate/deploy 
    modelsresults, its justthere for completements, you can use like this when you are workign with models 

    ctx.save_ouput("models","modelname") this will save the modelname into the models folder

    ifyoua re working with other types of uput from the other nodes if you dont specify the ctx.save_model("modelname") this gets saved inthe output folder,

    its iether you say you want it in the mdoels folder or you dont, (this seems redudant though it should be removed, )

    a usecase for this is this imagine you ahve images orvideas fromwherever maybe you took the videos or the images, you just upload them to s3 tosomerandombucket thn pass that bucket name intothe script 


    /output

    /logs ---> this is where allthe logsfromrunningthe applications are ported, in the vm we will have small agents running on each provisioned vm , 
    the agent will stream the logs fromthe vm tothe controllplaneandthen the cotroll plane ports thelogs tothe UI
    the agent willalsotrytocapture all thingsthat arebeingportedtothe stdout or sdterr and pass them to the controlplane 




the session id
the node




on recieving the request the ec2 choo






The specialized deploy node:
unlike the other nodes the deploy nodeis aspecialized nodeinthe sense that its stateful 
in this sense, when the user executes the deploy node, 
the system takes the model defined, itthen chatgpt it and provisdes an endpoint and creds the user can use to query their newly trained model, 

we now only support llms and regression models, 
when we hit the deploy node, we drop the current model and update itwiththe current one 