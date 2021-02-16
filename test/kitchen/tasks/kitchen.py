from invoke import task
from invoke.exceptions import Exit, ParseError

import glob
import json
import os.path
import re

@task
def genconfig(
    ctx,
    platform=None,
    provider="azure",
    osversions="all",
    testfiles=None,
    uservars=None,
    platformfile="platforms.json"
):
    """
    Create a kitchen config
    """
    if not platform:
        print("Must supply a platform to configure\n")
        raise Exit(1)

    if not testfiles:
        print("Must supply one or more testfiles to include\n")
        raise Exit(1)

    platforms = load_platforms(ctx, platformfile=platformfile)

    plat = platforms.get(platform)
    if not plat:
        print("Unknown platform {platform}.  Known platforms are {avail}\n".format(
            platform=platform,
            avail=list(platforms.keys())
        ))
        raise Exit(2)

    ## check to see if the OS is configured for the given provider
    prov = plat.get(provider)
    if not prov:
        print("Unknown provider {prov}.  Known providers for platform {plat} are {avail}\n".format(
            prov=provider,
            plat=platform,
            avail=list(plat.keys())
        ))
        raise Exit(3)

    ## get list of target OSes
    if osversions.lower() == "all":
        osversions = ".*"

    osimages = load_targets(ctx, prov, osversions)

    print("Chose os targets {}\n".format(osimages))

    # create the TEST_PLATFORMS environment variable
    testplatforms = ""
    for osimage in osimages:
        if testplatforms:
            testplatforms += "|"
        testplatforms += "{},{}".format(osimage, prov[osimage])

    print(testplatforms)

    # create the kitchen.yml file
    with open('tmpkitchen.yml', 'w') as kitchenyml:
        # first read the correct driver
        print("Adding driver file drivers/{}-driver.yml".format(provider))

        with open("drivers/{}-driver.yml".format(provider), 'r') as driverfile:
            for line in driverfile:
                kitchenyml.write(line)

        # read the generic contents
        with open("test-definitions/platforms-common.yml", 'r') as commonfile:
            for line in commonfile:
                kitchenyml.write(line)

        # now open the requested test files
        for f in glob.glob("test-definitions/{}.yml".format(testfiles)):
            if f.lower().endswith("platforms-common.yml"):
                print("Skipping common file")
            with open(f, 'r') as infile:
                print("Adding file {}\n".format(f))
                for line in infile:
                    kitchenyml.write(line)

    env = {}
    if uservars:
        env = load_user_env(ctx, provider, uservars)
    env['TEST_PLATFORMS'] = testplatforms
    ctx.run("erb tmpkitchen.yml > kitchen.yml", env=env)

def load_platforms(ctx, platformfile):
    with open(platformfile, "r") as f:
        platforms = json.load(f)
    return platforms

def load_targets(ctx, targethash, selections):
    returnlist = []
    commentpattern = re.compile("^comment")
    for selection in selections.split(","):
        selectionpattern = re.compile(selection)
        
        for key in targethash:
            if commentpattern.match(key):
                continue
            if selectionpattern.search(key):
                returnlist.append(key)
    return returnlist

def load_user_env(ctx, provider, varsfile):
    env = {}
    if os.path.exists(varsfile):
        with open("uservars.json", "r") as f:
            vars = json.load(f)
            for key, val in vars['global'].items():
                env[key] = val
            for key, val in vars[provider].items():
                env[key] = val
    return env
