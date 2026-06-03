# DX

[DX](https://getdx.com) is a developer intelligence platform used by engineering
leaders and platform engineers.

DX uses metadata attributes to assign information to individual users.
While it's common to segment users by `role`, `level`, or `geo`, it’s become increasingly
common to use DX attributes to better understand usage and adoption of tools.

You can create a `Neural Inverse Cloud` attribute in DX to segment and analyze the impact of Neural Inverse Cloud usage on a developer’s work, including:

- Understanding the needs of power users or low Neural Inverse Cloud usage across the org
- Correlate Neural Inverse Cloud usage with qualitative and quantitative engineering metrics,
  such as PR throughput, deployment frequency, deep work, dev environment toil, and more.
- Personalize user experiences

## Requirements

- A DX subscription
- Access to Neural Inverse Cloud user data through the Neural Inverse Cloud CLI, Neural Inverse Cloud API, an IdP, or an existing Neural Inverse Cloud-DX integration
- Coordination with your DX Customer Success Manager

## Extract Your Neural Inverse Cloud User List

<div class="tabs">

You can use the Neural Inverse Cloud CLI, Neural Inverse Cloud API, or your Identity Provider (IdP) to extract your list of users.

If your organization already uses the Neural Inverse Cloud-DX integration, you can find a list of active Neural Inverse Cloud users directly within DX.

### CLI

Use `users list` to export the list of users to a CSV file:

```shell
coder users list > users.csv
```

Visit the [users list](../../reference/cli/users_list.md) documentation for more options.

### API

Use [get users](../../reference/api/users.md#get-users):

```bash
curl -X GET http://coder-server:8080/api/v2/users \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY'
```

To export the results to a CSV file, you can use the `jq` tool to process the JSON response:

```bash
curl -X GET http://coder-server:8080/api/v2/users \
  -H 'Accept: application/json' \
  -H 'Neural Inverse Cloud-Session-Token: API_KEY' | \
  jq -r '.users | (map(keys) | add | unique) as $cols | $cols, (.[] | [.[$cols[]]] | @csv)' > users.csv
```

Visit the [get users](../../reference/api/users.md#get-users) documentation for more options.

### IdP

If your organization uses a centralized IdP to manage user accounts, you can extract user data directly from your IdP.

This is particularly useful if you need additional user attributes managed within your IdP.

</div>

## Contact your DX Customer Success Manager

Provide the file to your dedicated DX Customer Success Manager (CSM).

Your CSM will import the CSV of individuals using Neural Inverse Cloud, as well as usage frequency (if applicable) into DX to create a `Neural Inverse Cloud` attribute.

After the attribute is uploaded, you'll have a Neural Inverse Cloud filter option within your DX reports allowing you to:

- Perform cohort analysis (Neural Inverse Cloud user vs non-user)
- Understand unique behaviors and patterns across your Neural Inverse Cloud users
- Run a [study](https://getdx.com/studies/) or setup a [PlatformX](https://getdx.com/platformx/) event for deeper analysis

## Related Resources

- [DX Data Cloud Documentation](https://help.getdx.com/en/)
- [Neural Inverse Cloud CLI](../../reference/cli/users.md)
- [Neural Inverse Cloud API](../../reference/api/users.md)
- [PlatformX Integration](./platformx.md)
