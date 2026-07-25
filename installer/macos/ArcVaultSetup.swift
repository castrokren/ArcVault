import SwiftUI
import Foundation

@main
struct ArcVaultSetup: App {
    var body: some Scene {
        WindowGroup {
            SetupWizardView()
                .preferredColorScheme(.light)
        }
        .windowStyle(.hiddenTitleBar)
        .windowResizability(.contentMinSize)
    }
}

struct SetupWizardView: View {
    @State private var currentStep = 0
    @State private var selectedComponents = Set<String>()
    @State private var coordinatorPort = "8080"
    @State private var adminUsername = ""
    @State private var adminPassword = ""
    @State private var coordinatorURL = ""
    @State private var agentID = ""
    @State private var showingAlert = false
    @State private var alertMessage = ""

    var body: some View {
        VStack(spacing: 0) {
            // Header
            VStack {
                Image(systemName: "square.grid.2x2")
                    .font(.system(size: 48))
                    .foregroundColor(.blue)
                Text("ArcVault Setup")
                    .font(.title)
                    .fontWeight(.bold)
            }
            .padding()
            .frame(maxWidth: .infinity)
            .background(Color(nsColor: .controlBackgroundColor))

            // Content
            TabView(selection: $currentStep) {
                ComponentSelectionView(selectedComponents: $selectedComponents)
                    .tag(0)

                if selectedComponents.contains("coordinator") || selectedComponents.contains("both") {
                    CoordinatorConfigView(port: $coordinatorPort, username: $adminUsername, password: $adminPassword)
                        .tag(1)
                }

                if selectedComponents.contains("agent") || selectedComponents.contains("both") {
                    AgentConfigView(url: $coordinatorURL, agentID: $agentID)
                        .tag(2)
                }
            }
            .tabViewStyle(.page(indexDisplayMode: .never))
            .padding()

            // Navigation
            HStack {
                if currentStep > 0 {
                    Button("Back") {
                        currentStep -= 1
                    }
                    .keyboardShortcut(.cancelAction)
                }

                Spacer()

                if currentStep < (selectedComponents.isEmpty ? 0 : (selectedComponents.contains("both") ? 2 : 1)) {
                    Button("Next") {
                        currentStep += 1
                    }
                    .keyboardShortcut(.defaultAction)
                } else {
                    Button("Install") {
                        installArcVault()
                    }
                    .keyboardShortcut(.defaultAction)
                    .foregroundColor(.white)
                    .padding(8)
                    .background(Color.blue)
                    .cornerRadius(6)
                }
            }
            .padding()
        }
        .frame(minWidth: 500, minHeight: 400)
        .alert("Setup Result", isPresented: $showingAlert) {
            Button("OK") { NSApp.terminate(nil) }
        } message: {
            Text(alertMessage)
        }
    }

    private func installArcVault() {
        // This would call the setup wizard binary or write config files directly
        // For now, we simulate the installation
        alertMessage = "Setup completed! Services will start automatically."
        showingAlert = true

        // In a real implementation, this would:
        // 1. Write config.json files
        // 2. Load launchd services
        // 3. Open the dashboard URL
    }
}

struct ComponentSelectionView: View {
    @Binding var selectedComponents: Set<String>

    var body: some View {
        VStack(alignment: .leading, spacing: 20) {
            Text("Select Components")
                .font(.headline)

            Toggle(isOn: Binding(
                get: { selectedComponents.contains("coordinator") },
                set: { if $0 { selectedComponents.insert("coordinator") } else { selectedComponents.remove("coordinator") } }
            )) {
                VStack(alignment: .leading) {
                    Text("Coordinator (Server)")
                        .fontWeight(.semibold)
                    Text("Central server for managing backups")
                        .font(.caption)
                        .foregroundColor(.gray)
                }
            }

            Toggle(isOn: Binding(
                get: { selectedComponents.contains("agent") },
                set: { if $0 { selectedComponents.insert("agent") } else { selectedComponents.remove("agent") } }
            )) {
                VStack(alignment: .leading) {
                    Text("Agent (Client)")
                        .fontWeight(.semibold)
                    Text("Backup execution on this machine")
                        .font(.caption)
                        .foregroundColor(.gray)
                }
            }

            Spacer()
        }
        .padding()
    }
}

struct CoordinatorConfigView: View {
    @Binding var port: String
    @Binding var username: String
    @Binding var password: String

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("Coordinator Setup")
                .font(.headline)

            VStack(alignment: .leading) {
                Text("Port")
                    .font(.caption)
                    .foregroundColor(.gray)
                TextField("Port", text: $port)
                    .textFieldStyle(.roundedBorder)
            }

            VStack(alignment: .leading) {
                Text("Admin Username")
                    .font(.caption)
                    .foregroundColor(.gray)
                TextField("Username", text: $username)
                    .textFieldStyle(.roundedBorder)
            }

            VStack(alignment: .leading) {
                Text("Admin Password")
                    .font(.caption)
                    .foregroundColor(.gray)
                SecureField("Password", text: $password)
                    .textFieldStyle(.roundedBorder)
            }

            Spacer()
        }
        .padding()
    }
}

struct AgentConfigView: View {
    @Binding var url: String
    @Binding var agentID: String

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("Agent Setup")
                .font(.headline)

            VStack(alignment: .leading) {
                Text("Coordinator URL")
                    .font(.caption)
                    .foregroundColor(.gray)
                TextField("http://coordinator:8080", text: $url)
                    .textFieldStyle(.roundedBorder)
            }

            VStack(alignment: .leading) {
                Text("Agent ID")
                    .font(.caption)
                    .foregroundColor(.gray)
                TextField("Agent ID", text: $agentID)
                    .textFieldStyle(.roundedBorder)
            }

            Spacer()
        }
        .padding()
    }
}

#Preview {
    SetupWizardView()
}
