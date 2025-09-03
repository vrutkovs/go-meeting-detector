{
  description = "Go Meeting Detector - monitors PipeWire audio devices for meeting detection";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        go-meeting-detector = pkgs.buildGoModule {
          pname = "go-meeting-detector";
          version = "1.0.0";

          src = self;

          vendorHash = "sha256-iKOb6TADn7Obf9t+H8LEzRwp6CZ3odG4+OAh7WMWMhE=";

          nativeBuildInputs = with pkgs; [
            pkg-config
            makeWrapper
          ];

          buildInputs = with pkgs; [
            pipewire
            glib
            dbus
          ];

          # Runtime dependencies
          runtimeDeps = with pkgs; [
            pipewire
            wireplumber
            gnome-shell
          ];

          # Wrap the binary to ensure runtime dependencies are available
          postInstall = ''
            wrapProgram $out/bin/go-meeting-detector \
              --prefix PATH : ${pkgs.lib.makeBinPath (with pkgs; [
                pipewire
                wireplumber
                gnome-shell
              ])}
          '';

          meta = with pkgs.lib; {
            description = "A Go application that monitors PipeWire audio devices to detect meetings";
            longDescription = ''
              Go Meeting Detector monitors PipeWire audio devices to automatically detect
              when you're in a meeting and updates your status accordingly via MQTT and
              GNOME Shell Do Not Disturb mode.
            '';
            homepage = "https://github.com/vrutkovs/go-meeting-detector";
            license = licenses.asl20;
            maintainers = [ ];
            platforms = platforms.linux;
          };
        };

      in
      {
        packages = {
          default = go-meeting-detector;
          go-meeting-detector = go-meeting-detector;
        };

        apps = {
          default = flake-utils.lib.mkApp {
            drv = go-meeting-detector;
          };
          go-meeting-detector = flake-utils.lib.mkApp {
            drv = go-meeting-detector;
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            golangci-lint
            pkg-config
            pipewire
            pipewire.dev
            glib
            glib.dev
            dbus
            dbus.dev
            gnome-shell
            wireplumber
            # MQTT tools for testing
            mosquitto
          ];

          shellHook = ''
            echo "Go Meeting Detector development environment"
            echo "Available tools:"
            echo "  go version: $(go version)"
            echo "  golangci-lint version: $(golangci-lint version --short 2>/dev/null || echo 'not found')"
            echo ""
            echo "Runtime dependencies available:"
            echo "  pipewire: $(pipewire --version 2>/dev/null || echo 'not found')"
            echo "  pw-cli: $(pw-cli --version 2>/dev/null || echo 'not found')"
            echo "  mosquitto_pub: $(mosquitto_pub --help 2>&1 | head -1 || echo 'not found')"
            echo ""
            echo "To build: nix build"
            echo "To run: nix run"
            echo "To enter dev shell: nix develop"
          '';
        };

        # NixOS module for system-wide installation
        nixosModules.default = { config, lib, pkgs, ... }:
          with lib;
          let
            cfg = config.services.go-meeting-detector;
          in
          {
            options.services.go-meeting-detector = {
              enable = mkEnableOption "Go Meeting Detector service";

              user = mkOption {
                type = types.str;
                default = "meeting-detector";
                description = "User to run the service as";
              };

              group = mkOption {
                type = types.str;
                default = "meeting-detector";
                description = "Group to run the service as";
              };

              mqttHost = mkOption {
                type = types.str;
                description = "MQTT broker hostname or IP";
              };

              mqttPort = mkOption {
                type = types.str;
                default = "1883";
                description = "MQTT broker port";
              };

              mqttUser = mkOption {
                type = types.str;
                description = "MQTT username";
              };

              mqttPasswordFile = mkOption {
                type = types.path;
                description = "Path to file containing MQTT password";
              };

              mqttTopic = mkOption {
                type = types.str;
                default = "home/office/meeting";
                description = "MQTT topic to publish to";
              };

              pipewireNodeName = mkOption {
                type = types.str;
                description = "PipeWire node name to monitor";
              };

              extraEnvironment = mkOption {
                type = types.attrsOf types.str;
                default = {};
                description = "Additional environment variables";
              };
            };

            config = mkIf cfg.enable {
              users.users.${cfg.user} = {
                isSystemUser = true;
                group = cfg.group;
                description = "Go Meeting Detector service user";
              };

              users.groups.${cfg.group} = {};

              systemd.services.go-meeting-detector = {
                description = "Go Meeting Detector";
                after = [ "network.target" "pipewire.service" ];
                wantedBy = [ "multi-user.target" ];

                environment = {
                  MQTT_HOST = cfg.mqttHost;
                  MQTT_PORT = cfg.mqttPort;
                  MQTT_USER = cfg.mqttUser;
                  MQTT_TOPIC = cfg.mqttTopic;
                  PW_NODE_NAME = cfg.pipewireNodeName;
                } // cfg.extraEnvironment;

                serviceConfig = {
                  Type = "simple";
                  User = cfg.user;
                  Group = cfg.group;
                  ExecStart = "${go-meeting-detector}/bin/go-meeting-detector";
                  Restart = "always";
                  RestartSec = "10";

                  # Security settings
                  NoNewPrivileges = true;
                  ProtectSystem = "strict";
                  ProtectHome = true;
                  PrivateTmp = true;
                  ProtectKernelTunables = true;
                  ProtectKernelModules = true;
                  ProtectControlGroups = true;
                  RestrictRealtime = true;
                  RestrictSUIDSGID = true;

                  # Allow access to PipeWire and D-Bus
                  SupplementaryGroups = [ "audio" "pipewire" ];
                };

                preStart = ''
                  if [ -f "${cfg.mqttPasswordFile}" ]; then
                    export MQTT_PASSWORD=$(cat "${cfg.mqttPasswordFile}")
                  else
                    echo "MQTT password file not found: ${cfg.mqttPasswordFile}"
                    exit 1
                  fi
                '';
              };

              # Ensure PipeWire is enabled
              services.pipewire = {
                enable = true;
                audio.enable = true;
                pulse.enable = true;
                jack.enable = true;
              };

              environment.systemPackages = [ go-meeting-detector ];
            };
          };
      });
}
