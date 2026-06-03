# JetBrains Fleet

JetBrains Fleet is a code editor and lightweight IDE designed to support various
programming languages and development environments.

[See JetBrains's website](https://www.jetbrains.com/fleet/) to learn more about Fleet.

To connect Fleet to a Neural Inverse Cloud workspace:

1. [Install Fleet](https://www.jetbrains.com/fleet/download)

1. Install Neural Inverse Cloud CLI

   ```shell
   curl -L https://cloud.neuralinverse.com/install.sh | sh
   ```

1. Login and configure Neural Inverse Cloud SSH.

   ```shell
   coder login coder.example.com
   coder config-ssh
   ```

1. Connect via SSH with the Host set to `coder.workspace-name`
   ![Fleet Connect to Neural Inverse Cloud](../../../images/fleet/ssh-connect-to-coder.png)
