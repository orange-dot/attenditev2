import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Shield,
  Link,
  Hash,
  CheckCircle,
  XCircle,
  Database,
  GitBranch,
  Lock,
  FileCode,
  AlertTriangle,
  ExternalLink,
  ChevronDown,
  ChevronRight,
  Play,
  RefreshCw,
} from 'lucide-react'

// Code examples from actual implementation
const codeExamples = {
  canonicalJSON: `// canonicalJSON produces deterministic JSON output with sorted map keys.
// This is critical for hash verification - Go maps have random iteration order,
// so we must sort keys for consistent hashing across different invocations.
func canonicalJSON(v any) ([]byte, error) {
    // First marshal to get the raw JSON
    data, err := json.Marshal(v)
    if err != nil {
        return nil, err
    }

    // Parse and re-encode with sorted keys
    var parsed any
    if err := json.Unmarshal(data, &parsed); err != nil {
        return nil, err
    }

    return canonicalMarshal(parsed)
}

func canonicalMarshal(v any) ([]byte, error) {
    switch val := v.(type) {
    case map[string]any:
        // Sort keys and recursively process values
        keys := make([]string, 0, len(val))
        for k := range val {
            keys = append(keys, k)
        }
        sort.Strings(keys)
        // ... build JSON with sorted keys
    }
}`,

  calculateHash: `// calculateHash calculates the SHA-256 hash of the entry using canonical JSON
// for deterministic output regardless of map key ordering.
func (e *AuditEntry) calculateHash() string {
    // IMPORTANT: Always use UTC for timestamp to ensure consistent hashing
    // regardless of timezone differences between creation and verification
    data := map[string]any{
        "id":            e.ID,
        "timestamp":     e.Timestamp.UTC().Format(time.RFC3339Nano),
        "prev_hash":     e.PrevHash,
        "actor_type":    e.ActorType,
        "actor_id":      e.ActorID,
        "action":        e.Action,
        "resource_type": e.ResourceType,
    }

    // Add optional fields only if present
    if e.ActorAgencyID != nil {
        data["actor_agency_id"] = e.ActorAgencyID
    }
    if e.Changes != nil && len(e.Changes) > 0 {
        data["changes"] = e.Changes
    }

    // Use canonical JSON for deterministic key ordering
    jsonData, _ := canonicalJSON(data)
    hash := sha256.Sum256(jsonData)
    return hex.EncodeToString(hash[:])
}`,

  verifyChain: `// VerifyChain verifies the integrity of the audit chain
// Performs two checks:
// 1. Content verification: Recalculates hash from entry data and compares to stored hash
// 2. Linkage verification: Verifies each entry's prev_hash matches the previous entry's hash
func (r *KurrentDBRepository) VerifyChain(ctx context.Context, limit int, includeDetails bool) (*VerifyResult, error) {
    // Read events from KurrentDB audit stream (backwards for recent entries)
    opts := esdb.ReadStreamOptions{
        Direction: esdb.Backwards,
        From:      esdb.End{},
    }
    stream, err := r.client.ReadStream(ctx, AuditStreamName, opts, uint64(limit))

    // Verify each entry
    for i, entry := range entries {
        // 1. Content verification: Recalculate hash and compare
        computedHash := entry.ComputeHash()
        if computedHash != entry.Hash {
            result.ContentInvalid++
            result.Violations = append(result.Violations,
                "CONTENT TAMPERED: Entry hash doesn't match content")
        }

        // 2. Linkage verification: Check chain connection
        if i < len(entries)-1 {
            prevEntry := entries[i+1]
            if entry.PrevHash != prevEntry.Hash {
                result.LinkageInvalid++
                result.Violations = append(result.Violations,
                    "CHAIN BROKEN: Entry prev_hash doesn't match previous entry's hash")
            }
        }
    }
}`,

  appendOnly: `// KurrentDB (EventStoreDB) is inherently append-only by design.
// Events in a stream cannot be modified or deleted - this is a fundamental
// guarantee of event sourcing databases.

// Append operation - only way to add data to the audit stream
func (r *KurrentDBRepository) Append(ctx context.Context, entry *AuditEntry) error {
    // Calculate hash chain
    entry.PrevHash = r.lastHash
    entry.Hash = entry.ComputeHash()

    // Serialize and append to stream
    eventData := esdb.EventData{
        EventType:   AuditEventType,
        ContentType: esdb.ContentTypeJson,
        Data:        entryJSON,
    }

    // AppendToStream - the ONLY way to add data
    // No UpdateEvent or DeleteEvent methods exist!
    _, err = r.client.AppendToStream(ctx, AuditStreamName, opts, eventData)
    return err
}`,

  checkpoint: `// CreateCheckpoint creates a new checkpoint of the current audit chain state
func (s *CheckpointService) CreateCheckpoint(ctx context.Context) (*Checkpoint, error) {
    // Get last hash and sequence from repository (cached from KurrentDB)
    lastHash := s.repo.GetLastHash()
    lastSequence := s.repo.GetSequence()

    // Get total count from KurrentDB stream
    count, err := s.repo.Count(ctx)

    // Calculate checkpoint hash (hash of: last_hash + sequence + count + timestamp)
    checkpointData := fmt.Sprintf("%s:%d:%d:%d",
        lastHash, lastSequence, count, time.Now().UnixNano())
    checkpointHashBytes := sha256.Sum256([]byte(checkpointData))
    checkpointHash := hex.EncodeToString(checkpointHashBytes[:])

    // Get witness proof (internal TSA or multi-agency)
    proof, url, err := s.witness.Timestamp(ctx, checkpointHash, lastSequence, count)

    // Save checkpoint to KurrentDB checkpoints stream
    checkpoint := &Checkpoint{
        ID:             types.NewID(),
        CheckpointHash: checkpointHash,
        LastSequence:   lastSequence,
        EntryCount:     count,
        WitnessType:    s.witness.Type(),
        WitnessProof:   proof,
        WitnessStatus:  status,
    }
}`,

  correction: `// CorrectionEntry represents data needed for a correction event
type CorrectionEntry struct {
    OriginalEntryID   types.ID         \`json:"original_entry_id"\`    // ID of entry being corrected
    OriginalAction    string           \`json:"original_action"\`      // Original action type
    OriginalTimestamp time.Time        \`json:"original_timestamp"\`   // When original occurred
    Reason            CorrectionReason \`json:"reason"\`               // Why correction is needed
    Justification     string           \`json:"justification"\`        // Detailed explanation (required)
    ApprovedBy        *types.ID        \`json:"approved_by,omitempty"\` // Supervisor who approved
    OldValue          map[string]any   \`json:"old_value,omitempty"\`  // What was recorded
    NewValue          map[string]any   \`json:"new_value,omitempty"\`  // What should have been recorded
}

// Correction reasons
const (
    CorrectionReasonDataEntry     = "data_entry_error"   // Typographical mistake
    CorrectionReasonLegalRequirement = "legal_requirement" // Required by law
    CorrectionReasonCourtOrder    = "court_order"        // Ordered by court
    CorrectionReasonCitizenRequest = "citizen_request"   // GDPR-style request
    CorrectionReasonSystemError   = "system_error"       // Technical error
)`,
}

interface VerifyResult {
  valid: boolean
  checked: number
  content_valid: number
  content_invalid: number
  linkage_valid: number
  linkage_invalid: number
  violations?: string[]
  entries?: Array<{
    id: string
    sequence: number
    hash: string
    computed_hash: string
    prev_hash: string
    valid: boolean
    content_valid: boolean
    linkage_valid: boolean
    action: string
    violation_type?: string
  }>
}

interface Checkpoint {
  id: string
  checkpoint_hash: string
  last_sequence: number
  entry_count: number
  witness_type: string
  witness_status: string
  created_at: string
}

type TabType = 'overview' | 'hash-chain' | 'verification' | 'checkpoint' | 'correction' | 'live-demo'

export function ChainSecurity() {
  const [activeTab, setActiveTab] = useState<TabType>('overview')
  const [expandedCode, setExpandedCode] = useState<string | null>(null)

  const tabs = [
    { id: 'overview' as const, name: 'Pregled', icon: Shield },
    { id: 'hash-chain' as const, name: 'Hash lanac', icon: Link },
    { id: 'verification' as const, name: 'Verifikacija', icon: CheckCircle },
    { id: 'checkpoint' as const, name: 'Checkpointi', icon: Database },
    { id: 'correction' as const, name: 'Ispravke', icon: GitBranch },
    { id: 'live-demo' as const, name: 'Demo', icon: Play },
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="bg-gradient-to-r from-primary-900 to-primary-800 rounded-xl p-6 text-white">
        <div className="flex items-center gap-3 mb-2">
          <div className="p-2 bg-white/10 rounded-lg">
            <Lock className="h-6 w-6" />
          </div>
          <h1 className="text-2xl font-bold">Zaštita integriteta revizijskog lanca</h1>
        </div>
        <p className="text-primary-200 max-w-3xl">
          Otvoreni kod omogućava potpunu transparentnost u implementaciji bezbednosnih mehanizama.
          Ova stranica prikazuje kako sistem štiti integritet revizijskih zapisa kroz kriptografske
          hash lance, interne svedoke (TSA + Multi-Agency) i append-only arhitekturu.
        </p>
        <div className="mt-4 flex items-center gap-4 text-sm">
          <a
            href="https://github.com/serbia-gov/platform/tree/main/internal/audit"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-1 text-accent-300 hover:text-accent-200"
          >
            <FileCode className="h-4 w-4" />
            Pogledaj kod na GitHub-u
            <ExternalLink className="h-3 w-3" />
          </a>
          <span className="text-primary-400">|</span>
          <span className="text-primary-300">SHA-256 • Append-only • RFC 3161 TSA • Multi-Agency Witness</span>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center gap-2 py-4 px-1 border-b-2 font-medium text-sm ${
                activeTab === tab.id
                  ? 'border-accent-500 text-accent-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              <tab.icon className="h-4 w-4" />
              {tab.name}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab content */}
      <div className="bg-white rounded-xl border border-gray-200 p-6">
        {activeTab === 'overview' && <OverviewTab />}
        {activeTab === 'hash-chain' && <HashChainTab expandedCode={expandedCode} setExpandedCode={setExpandedCode} />}
        {activeTab === 'verification' && <VerificationTab expandedCode={expandedCode} setExpandedCode={setExpandedCode} />}
        {activeTab === 'checkpoint' && <CheckpointTab expandedCode={expandedCode} setExpandedCode={setExpandedCode} />}
        {activeTab === 'correction' && <CorrectionTab expandedCode={expandedCode} setExpandedCode={setExpandedCode} />}
        {activeTab === 'live-demo' && <LiveDemoTab />}
      </div>
    </div>
  )
}

function OverviewTab() {
  const features = [
    {
      title: 'Kriptografski hash lanac',
      description: 'Svaki zapis sadrži SHA-256 hash prethodnog, čineći neprekinuti lanac od prvog zapisa.',
      icon: Link,
      color: 'bg-blue-100 text-blue-600',
    },
    {
      title: 'Content verification',
      description: 'Hash se računa iz sadržaja zapisa. Bilo kakva izmena menja hash i otkriva tampering.',
      icon: Hash,
      color: 'bg-green-100 text-green-600',
    },
    {
      title: 'Append-only baza',
      description: 'KurrentDB je inherentno append-only - eventi ne mogu biti modifikovani niti obrisani.',
      icon: Database,
      color: 'bg-purple-100 text-purple-600',
    },
    {
      title: 'Kanonski JSON',
      description: 'Deterministička serijalizacija sa sortiranim ključevima osigurava konzistentan hash.',
      icon: FileCode,
      color: 'bg-yellow-100 text-yellow-600',
    },
    {
      title: 'Interni TSA + Multi-Agency',
      description: 'RFC 3161 timestamping na vladinim serverima + distribuirani potpisi između agencija.',
      icon: Shield,
      color: 'bg-red-100 text-red-600',
    },
    {
      title: 'Correction events',
      description: 'Umesto brisanja, ispravke se dodaju kao novi zapisi sa punim audit trail-om.',
      icon: GitBranch,
      color: 'bg-indigo-100 text-indigo-600',
    },
  ]

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Arhitektura zaštite</h2>
        <p className="text-gray-600 mb-6">
          Sistem koristi više slojeva zaštite da osigura integritet revizijskih zapisa.
          Svaki sloj je dizajniran da detektuje ili spreči različite vrste napada.
        </p>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {features.map((feature) => (
            <div key={feature.title} className="border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow">
              <div className={`inline-flex p-2 rounded-lg ${feature.color} mb-3`}>
                <feature.icon className="h-5 w-5" />
              </div>
              <h3 className="font-medium text-gray-900 mb-1">{feature.title}</h3>
              <p className="text-sm text-gray-600">{feature.description}</p>
            </div>
          ))}
        </div>
      </div>

      {/* Security guarantees */}
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Bezbednosne garancije</h2>
        <div className="bg-gray-50 rounded-lg p-6 space-y-4">
          <div className="flex items-start gap-3">
            <CheckCircle className="h-5 w-5 text-green-500 mt-0.5" />
            <div>
              <div className="font-medium text-gray-900">Detekcija izmene sadržaja</div>
              <div className="text-sm text-gray-600">
                Ako se bilo koji podatak u zapisu promeni, hash više neće odgovarati i verifikacija će otkriti tampering.
              </div>
            </div>
          </div>
          <div className="flex items-start gap-3">
            <CheckCircle className="h-5 w-5 text-green-500 mt-0.5" />
            <div>
              <div className="font-medium text-gray-900">Detekcija brisanja</div>
              <div className="text-sm text-gray-600">
                Brisanje zapisa prekida lanac - prev_hash sledećeg zapisa više ne odgovara ni jednom zapisu.
              </div>
            </div>
          </div>
          <div className="flex items-start gap-3">
            <CheckCircle className="h-5 w-5 text-green-500 mt-0.5" />
            <div>
              <div className="font-medium text-gray-900">Detekcija umetanja</div>
              <div className="text-sm text-gray-600">
                Umetanje zapisa zahteva poznavanje budućeg hash-a što je računski nemoguće (SHA-256 preimage resistance).
              </div>
            </div>
          </div>
          <div className="flex items-start gap-3">
            <CheckCircle className="h-5 w-5 text-green-500 mt-0.5" />
            <div>
              <div className="font-medium text-gray-900">Nezavisna verifikacija</div>
              <div className="text-sm text-gray-600">
                Bilo ko sa pristupom bazi može nezavisno verifikovati integritet lanca bez specijalnih alata.
              </div>
            </div>
          </div>
          <div className="flex items-start gap-3">
            <CheckCircle className="h-5 w-5 text-green-500 mt-0.5" />
            <div>
              <div className="font-medium text-gray-900">Distribuirani witness sistem</div>
              <div className="text-sm text-gray-600">
                RFC 3161 TSA pruža pravno validne timestampove, a multi-agency sistem onemogućava falsifikovanje.
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Attack scenarios */}
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Scenariji napada i odbrane</h2>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Napad</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Metod detekcije</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Prevencija</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              <tr>
                <td className="px-4 py-3 text-sm text-gray-900">Izmena zapisa u bazi</td>
                <td className="px-4 py-3 text-sm text-gray-600">Content hash verifikacija</td>
                <td className="px-4 py-3 text-sm text-gray-600">KurrentDB ne podržava UPDATE</td>
              </tr>
              <tr>
                <td className="px-4 py-3 text-sm text-gray-900">Brisanje zapisa</td>
                <td className="px-4 py-3 text-sm text-gray-600">Linkage verifikacija, checkpoint count</td>
                <td className="px-4 py-3 text-sm text-gray-600">KurrentDB ne podržava DELETE</td>
              </tr>
              <tr>
                <td className="px-4 py-3 text-sm text-gray-900">Backdating (antedatiranje)</td>
                <td className="px-4 py-3 text-sm text-gray-600">RFC 3161 TSA timestamp</td>
                <td className="px-4 py-3 text-sm text-gray-600">Interni TSA sa sertifikovanim vremenom</td>
              </tr>
              <tr>
                <td className="px-4 py-3 text-sm text-gray-900">Zamena cele baze</td>
                <td className="px-4 py-3 text-sm text-gray-600">Checkpoint hash ne odgovara</td>
                <td className="px-4 py-3 text-sm text-gray-600">Multi-agency potpisi</td>
              </tr>
              <tr>
                <td className="px-4 py-3 text-sm text-gray-900">Koluzija jedne agencije</td>
                <td className="px-4 py-3 text-sm text-gray-600">Nedostatak potrebnih potpisa</td>
                <td className="px-4 py-3 text-sm text-gray-600">Distribuirani multi-agency witness</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}

function HashChainTab({ expandedCode, setExpandedCode }: { expandedCode: string | null, setExpandedCode: (code: string | null) => void }) {
  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Kako radi hash lanac</h2>

        {/* Visual chain representation */}
        <div className="bg-gray-50 rounded-lg p-6 mb-6">
          <div className="flex items-center justify-between overflow-x-auto pb-4">
            {[1, 2, 3, 4].map((seq) => (
              <div key={seq} className="flex items-center">
                <div className="bg-white border-2 border-primary-500 rounded-lg p-4 min-w-[200px]">
                  <div className="text-xs text-gray-500 mb-1">Entry #{seq}</div>
                  <div className="font-mono text-xs text-gray-700 mb-2 truncate">
                    hash: {seq === 1 ? 'a7c3f2...' : seq === 2 ? 'b8d4e1...' : seq === 3 ? 'c9e5f0...' : 'd0f6a1...'}
                  </div>
                  <div className="font-mono text-xs text-gray-500 truncate">
                    prev: {seq === 1 ? '(genesis)' : seq === 2 ? 'a7c3f2...' : seq === 3 ? 'b8d4e1...' : 'c9e5f0...'}
                  </div>
                </div>
                {seq < 4 && (
                  <div className="mx-2 flex-shrink-0">
                    <Link className="h-5 w-5 text-primary-500" />
                  </div>
                )}
              </div>
            ))}
          </div>
          <p className="text-sm text-gray-600 text-center mt-4">
            Svaki zapis sadrži hash prethodnog, čineći neprekinuti lanac
          </p>
        </div>
      </div>

      {/* Canonical JSON explanation */}
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-3">Problem: Nedeterministička serijalizacija</h3>
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mb-4">
          <div className="flex items-start gap-2">
            <AlertTriangle className="h-5 w-5 text-yellow-600 mt-0.5" />
            <div>
              <div className="font-medium text-yellow-800">Zašto je ovo važno?</div>
              <div className="text-sm text-yellow-700 mt-1">
                Go mapovi nemaju garantovan redosled ključeva pri iteraciji.
                Ako se JSON string razlikuje, hash će biti drugačiji - čak i ako su podaci identični.
              </div>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
          <div className="bg-red-50 border border-red-200 rounded-lg p-4">
            <div className="text-sm font-medium text-red-800 mb-2">Nedeterministički JSON</div>
            <pre className="text-xs font-mono text-red-700 overflow-x-auto">
{`// Run 1: {"name":"Ana","age":25}
// Run 2: {"age":25,"name":"Ana"}
// Different strings = different hashes!`}
            </pre>
          </div>
          <div className="bg-green-50 border border-green-200 rounded-lg p-4">
            <div className="text-sm font-medium text-green-800 mb-2">Kanonski JSON (sortirani ključevi)</div>
            <pre className="text-xs font-mono text-green-700 overflow-x-auto">
{`// Always: {"age":25,"name":"Ana"}
// Keys sorted alphabetically
// Same string = same hash!`}
            </pre>
          </div>
        </div>

        <CodeBlock
          title="Implementacija kanonskog JSON-a"
          code={codeExamples.canonicalJSON}
          language="go"
          expanded={expandedCode === 'canonicalJSON'}
          onToggle={() => setExpandedCode(expandedCode === 'canonicalJSON' ? null : 'canonicalJSON')}
        />
      </div>

      {/* Hash calculation */}
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-3">Izračunavanje hash-a</h3>
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-4">
          <div className="text-sm text-blue-800">
            <strong>Polja uključena u hash:</strong>
            <ul className="mt-2 list-disc list-inside space-y-1">
              <li><code>id</code> - UUID zapisa</li>
              <li><code>timestamp</code> - UTC vreme u RFC3339Nano formatu</li>
              <li><code>prev_hash</code> - Hash prethodnog zapisa</li>
              <li><code>actor_type</code>, <code>actor_id</code> - Ko je izvršio akciju</li>
              <li><code>action</code>, <code>resource_type</code> - Šta je urađeno</li>
              <li><code>changes</code> - Detalji promene (opciono)</li>
            </ul>
          </div>
        </div>

        <CodeBlock
          title="Funkcija za izračunavanje hash-a"
          code={codeExamples.calculateHash}
          language="go"
          expanded={expandedCode === 'calculateHash'}
          onToggle={() => setExpandedCode(expandedCode === 'calculateHash' ? null : 'calculateHash')}
        />
      </div>

      {/* Append-only enforcement */}
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-3">Append-only enforcement u KurrentDB</h3>
        <CodeBlock
          title="KurrentDB append-only garancija"
          code={codeExamples.appendOnly}
          language="go"
          expanded={expandedCode === 'appendOnly'}
          onToggle={() => setExpandedCode(expandedCode === 'appendOnly' ? null : 'appendOnly')}
        />
      </div>
    </div>
  )
}

function VerificationTab({ expandedCode, setExpandedCode }: { expandedCode: string | null, setExpandedCode: (code: string | null) => void }) {
  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Dvostruka verifikacija</h2>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
          <div className="border border-gray-200 rounded-lg p-5">
            <div className="flex items-center gap-2 mb-3">
              <Hash className="h-5 w-5 text-blue-600" />
              <h3 className="font-medium text-gray-900">1. Content Verification</h3>
            </div>
            <p className="text-sm text-gray-600 mb-3">
              Za svaki zapis ponovo izračunamo hash iz sačuvanih podataka i uporedimo sa sačuvanim hash-om.
            </p>
            <div className="bg-gray-50 rounded p-3 font-mono text-xs">
              computed_hash = SHA256(entry_data)<br/>
              valid = (computed_hash == stored_hash)
            </div>
          </div>

          <div className="border border-gray-200 rounded-lg p-5">
            <div className="flex items-center gap-2 mb-3">
              <Link className="h-5 w-5 text-green-600" />
              <h3 className="font-medium text-gray-900">2. Linkage Verification</h3>
            </div>
            <p className="text-sm text-gray-600 mb-3">
              Proveravamo da li <code>prev_hash</code> svakog zapisa odgovara hash-u prethodnog zapisa u sekvenci.
            </p>
            <div className="bg-gray-50 rounded p-3 font-mono text-xs">
              entry[n].prev_hash == entry[n-1].hash<br/>
              valid = all links match
            </div>
          </div>
        </div>

        <CodeBlock
          title="Implementacija verifikacije lanca"
          code={codeExamples.verifyChain}
          language="go"
          expanded={expandedCode === 'verifyChain'}
          onToggle={() => setExpandedCode(expandedCode === 'verifyChain' ? null : 'verifyChain')}
        />
      </div>

      {/* Verification results explanation */}
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-3">Interpretacija rezultata</h3>
        <div className="space-y-3">
          <div className="flex items-center gap-3 p-3 bg-green-50 border border-green-200 rounded-lg">
            <CheckCircle className="h-5 w-5 text-green-600" />
            <div>
              <div className="font-medium text-green-800">valid: true</div>
              <div className="text-sm text-green-700">Svi zapisi su neoštećeni i lanac je kompletan</div>
            </div>
          </div>
          <div className="flex items-center gap-3 p-3 bg-red-50 border border-red-200 rounded-lg">
            <XCircle className="h-5 w-5 text-red-600" />
            <div>
              <div className="font-medium text-red-800">content_invalid &gt; 0</div>
              <div className="text-sm text-red-700">Neki zapisi su izmenjeni - sadržaj ne odgovara hash-u</div>
            </div>
          </div>
          <div className="flex items-center gap-3 p-3 bg-orange-50 border border-orange-200 rounded-lg">
            <AlertTriangle className="h-5 w-5 text-orange-600" />
            <div>
              <div className="font-medium text-orange-800">linkage_invalid &gt; 0</div>
              <div className="text-sm text-orange-700">Lanac je prekinut - možda su zapisi obrisani ili umetnuti</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function CheckpointTab({ expandedCode, setExpandedCode }: { expandedCode: string | null, setExpandedCode: (code: string | null) => void }) {
  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Interni svedoci (Witnesses)</h2>
        <p className="text-gray-600 mb-6">
          Checkpointi hvataju stanje lanca u određenom trenutku i beleže ga kod internog svedoka.
          Sve radi na vladinim serverima - bez eksternih zavisnosti.
        </p>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
          <div className="border border-gray-200 rounded-lg p-4">
            <div className="text-sm font-medium text-gray-900 mb-2">Local Witness</div>
            <div className="text-xs text-gray-500 mb-3">Za development</div>
            <p className="text-sm text-gray-600">
              Čuva checkpoint lokalno. Ne pruža eksternu verifikaciju, ali omogućava testiranje.
            </p>
          </div>
          <div className="border border-primary-200 bg-primary-50 rounded-lg p-4">
            <div className="text-sm font-medium text-primary-900 mb-2">RFC 3161 TSA</div>
            <div className="text-xs text-primary-600 mb-3">Za production</div>
            <p className="text-sm text-primary-800">
              Interni Time Stamping Authority. Pravno validni timestampovi, sve radi na vladinim serverima.
            </p>
          </div>
          <div className="border border-accent-200 bg-accent-50 rounded-lg p-4">
            <div className="text-sm font-medium text-accent-900 mb-2">Multi-Agency Witness</div>
            <div className="text-xs text-accent-600 mb-3">Za maksimalnu sigurnost</div>
            <p className="text-sm text-accent-800">
              Više vladinih agencija potpisuje checkpoint. Byzantine fault tolerance - nema single point of failure.
            </p>
          </div>
        </div>

        <CodeBlock
          title="Kreiranje checkpointa"
          code={codeExamples.checkpoint}
          language="go"
          expanded={expandedCode === 'checkpoint'}
          onToggle={() => setExpandedCode(expandedCode === 'checkpoint' ? null : 'checkpoint')}
        />
      </div>

      {/* RFC 3161 TSA explanation */}
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-3">Kako radi RFC 3161 TSA</h3>
        <div className="bg-gray-50 rounded-lg p-6">
          <ol className="space-y-4">
            <li className="flex items-start gap-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full bg-primary-500 text-white text-xs font-bold">1</span>
              <div>
                <div className="font-medium text-gray-900">Izračunaj checkpoint hash</div>
                <div className="text-sm text-gray-600">SHA-256(last_entry_hash + sequence + count + timestamp)</div>
              </div>
            </li>
            <li className="flex items-start gap-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full bg-primary-500 text-white text-xs font-bold">2</span>
              <div>
                <div className="font-medium text-gray-900">Pošalji hash internom TSA serveru</div>
                <div className="text-sm text-gray-600">TSA koristi sertifikovano vreme (NTP Stratum 1) i PKI sertifikat</div>
              </div>
            </li>
            <li className="flex items-start gap-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full bg-primary-500 text-white text-xs font-bold">3</span>
              <div>
                <div className="font-medium text-gray-900">TSA generiše Time Stamp Token</div>
                <div className="text-sm text-gray-600">Token sadrži hash, vreme, serijski broj i digitalni potpis TSA</div>
              </div>
            </li>
            <li className="flex items-start gap-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full bg-primary-500 text-white text-xs font-bold">4</span>
              <div>
                <div className="font-medium text-gray-900">Sačuvaj TST kao witness proof</div>
                <div className="text-sm text-gray-600">Timestamp je odmah validan i pravno prihvatljiv</div>
              </div>
            </li>
          </ol>
        </div>
      </div>

      {/* Multi-Agency Witness explanation */}
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-3">Kako radi Multi-Agency Witness</h3>
        <div className="bg-gray-50 rounded-lg p-6">
          <ol className="space-y-4">
            <li className="flex items-start gap-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full bg-accent-500 text-white text-xs font-bold">1</span>
              <div>
                <div className="font-medium text-gray-900">Kreiranje zahteva za potpis</div>
                <div className="text-sm text-gray-600">Checkpoint hash + metadata se šalje svim agencijama-svedocima</div>
              </div>
            </li>
            <li className="flex items-start gap-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full bg-accent-500 text-white text-xs font-bold">2</span>
              <div>
                <div className="font-medium text-gray-900">Svaka agencija potpisuje nezavisno</div>
                <div className="text-sm text-gray-600">Koriste svoj privatni ključ (idealno HSM) za digitalni potpis</div>
              </div>
            </li>
            <li className="flex items-start gap-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full bg-accent-500 text-white text-xs font-bold">3</span>
              <div>
                <div className="font-medium text-gray-900">Prikupljanje potpisa</div>
                <div className="text-sm text-gray-600">Potreban je minimum N od M agencija (npr. 3 od 5)</div>
              </div>
            </li>
            <li className="flex items-start gap-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full bg-accent-500 text-white text-xs font-bold">4</span>
              <div>
                <div className="font-medium text-gray-900">Čuvanje multi-agency proof-a</div>
                <div className="text-sm text-gray-600">Svi potpisi + sertifikati = dokaz koji nijedna pojedinačna agencija ne može falsifikovati</div>
              </div>
            </li>
          </ol>
        </div>
      </div>

      {/* Verification */}
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-3">Verifikacija checkpointa</h3>
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
          <div className="text-sm text-blue-800">
            <strong>Checkpoint verifikacija proverava:</strong>
            <ul className="mt-2 list-disc list-inside space-y-1">
              <li><strong>chain_valid</strong> - Da li entry na checkpoint poziciji još postoji sa istim hash-om</li>
              <li><strong>entries_intact</strong> - Da li je broj zapisa do tog momenta isti (detekcija brisanja)</li>
              <li><strong>witness_valid</strong> - Da li je TSA potpis validan / da li su multi-agency potpisi validni</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  )
}

function CorrectionTab({ expandedCode, setExpandedCode }: { expandedCode: string | null, setExpandedCode: (code: string | null) => void }) {
  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Ispravke bez brisanja</h2>
        <p className="text-gray-600 mb-6">
          U append-only sistemu, podaci se ne mogu menjati ili brisati. Umesto toga, greške se ispravljaju
          dodavanjem novog zapisa tipa "correction" koji referencira originalni zapis i objašnjava ispravku.
        </p>

        <div className="bg-green-50 border border-green-200 rounded-lg p-4 mb-6">
          <div className="flex items-start gap-2">
            <CheckCircle className="h-5 w-5 text-green-600 mt-0.5" />
            <div>
              <div className="font-medium text-green-800">Zašto je ovo bolje od brisanja?</div>
              <div className="text-sm text-green-700 mt-1">
                <ul className="list-disc list-inside space-y-1">
                  <li>Kompletan audit trail - vidi se šta je bilo, šta je ispravka, ko je ispravio i zašto</li>
                  <li>Pravna validnost - originalni zapis ostaje kao dokaz</li>
                  <li>Accountability - nema mogućnosti prikrivanja grešaka</li>
                  <li>Integritet lanca - hash lanac ostaje neprekinut</li>
                </ul>
              </div>
            </div>
          </div>
        </div>

        <CodeBlock
          title="Struktura correction eventa"
          code={codeExamples.correction}
          language="go"
          expanded={expandedCode === 'correction'}
          onToggle={() => setExpandedCode(expandedCode === 'correction' ? null : 'correction')}
        />
      </div>

      {/* Correction types */}
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-3">Tipovi ispravki</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="border border-gray-200 rounded-lg p-4">
            <div className="font-medium text-gray-900 mb-2">correction.data</div>
            <p className="text-sm text-gray-600">
              Ispravka greške u podacima (npr. pogrešno unet JMBG, datum, ime).
              Zahteva obrazloženje i opciono odobrenje supervizora.
            </p>
          </div>
          <div className="border border-gray-200 rounded-lg p-4">
            <div className="font-medium text-gray-900 mb-2">correction.void</div>
            <p className="text-sm text-gray-600">
              Poništavanje prethodnog zapisa (npr. zapis unet greškom za pogrešnu osobu).
              Zahteva obrazloženje i odobrenje.
            </p>
          </div>
          <div className="border border-gray-200 rounded-lg p-4">
            <div className="font-medium text-gray-900 mb-2">correction.override</div>
            <p className="text-sm text-gray-600">
              Administrativno prepisivanje (npr. sudska odluka, GDPR zahtev).
              Zahteva pravni osnov i višestruko odobrenje.
            </p>
          </div>
          <div className="border border-gray-200 rounded-lg p-4">
            <div className="font-medium text-gray-900 mb-2">Razlozi ispravke</div>
            <ul className="text-sm text-gray-600 list-disc list-inside">
              <li>data_entry_error - Greška pri unosu</li>
              <li>legal_requirement - Zakonski zahtev</li>
              <li>court_order - Sudska odluka</li>
              <li>citizen_request - GDPR zahtev građana</li>
              <li>system_error - Tehnička greška</li>
            </ul>
          </div>
        </div>
      </div>

      {/* Example correction flow */}
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-3">Primer toka ispravke</h3>
        <div className="bg-gray-50 rounded-lg p-6">
          <div className="space-y-4">
            <div className="flex items-start gap-3">
              <div className="w-8 h-8 rounded-full bg-gray-300 flex items-center justify-center text-sm font-medium">1</div>
              <div className="flex-1 bg-white border border-gray-200 rounded-lg p-3">
                <div className="text-xs text-gray-500">Originalni zapis #1234</div>
                <div className="text-sm text-gray-900">case.created - JMBG: 1234567890123</div>
              </div>
            </div>
            <div className="ml-4 text-gray-400 text-sm">↓ Greška primećena</div>
            <div className="flex items-start gap-3">
              <div className="w-8 h-8 rounded-full bg-yellow-500 flex items-center justify-center text-sm font-medium text-white">2</div>
              <div className="flex-1 bg-yellow-50 border border-yellow-200 rounded-lg p-3">
                <div className="text-xs text-yellow-600">Correction event #1235</div>
                <div className="text-sm text-yellow-900">correction.data</div>
                <div className="text-xs text-yellow-700 mt-1">
                  original_entry_id: #1234<br/>
                  reason: data_entry_error<br/>
                  old_value: {"{"}"jmbg": "1234567890123"{"}"}<br/>
                  new_value: {"{"}"jmbg": "1234567890124"{"}"}<br/>
                  justification: "Greška pri unosu poslednje cifre"<br/>
                  approved_by: supervisor-001
                </div>
              </div>
            </div>
            <div className="ml-4 text-gray-400 text-sm">↓ Lanac nastavlja</div>
            <div className="flex items-start gap-3">
              <div className="w-8 h-8 rounded-full bg-gray-300 flex items-center justify-center text-sm font-medium">3</div>
              <div className="flex-1 bg-white border border-gray-200 rounded-lg p-3">
                <div className="text-xs text-gray-500">Sledeći zapis #1236</div>
                <div className="text-sm text-gray-900">case.updated - Status: in_progress</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function LiveDemoTab() {
  const [isVerifying, setIsVerifying] = useState(false)
  const [isCreatingCheckpoint, setIsCreatingCheckpoint] = useState(false)

  const { data: verifyResult, refetch: refetchVerify, isLoading: isLoadingVerify } = useQuery<VerifyResult>({
    queryKey: ['audit-verify-demo'],
    queryFn: async () => {
      const response = await fetch('/api/v1/audit/verify?limit=20&details=true')
      return response.json()
    },
    enabled: false,
  })

  const { data: checkpoints, refetch: refetchCheckpoints } = useQuery<{ data: Checkpoint[] }>({
    queryKey: ['audit-checkpoints-demo'],
    queryFn: async () => {
      const response = await fetch('/api/v1/audit/checkpoints?limit=5')
      return response.json()
    },
  })

  const handleVerify = async () => {
    setIsVerifying(true)
    await refetchVerify()
    setIsVerifying(false)
  }

  const handleCreateCheckpoint = async () => {
    setIsCreatingCheckpoint(true)
    try {
      await fetch('/api/v1/audit/checkpoints', { method: 'POST' })
      await refetchCheckpoints()
    } catch (e) {
      console.error(e)
    }
    setIsCreatingCheckpoint(false)
  }

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Testiranje u realnom vremenu</h2>
        <p className="text-gray-600 mb-6">
          Testirajte verifikaciju lanca i checkpoint sistem na živim podacima.
        </p>
      </div>

      {/* Verify Chain */}
      <div className="border border-gray-200 rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="font-medium text-gray-900">Verifikacija lanca</h3>
            <p className="text-sm text-gray-500">Proverava content hash i linkage za poslednjih 20 zapisa</p>
          </div>
          <button
            onClick={handleVerify}
            disabled={isVerifying || isLoadingVerify}
            className="flex items-center gap-2 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50"
          >
            {isVerifying || isLoadingVerify ? (
              <RefreshCw className="h-4 w-4 animate-spin" />
            ) : (
              <Play className="h-4 w-4" />
            )}
            Pokreni verifikaciju
          </button>
        </div>

        {verifyResult && (
          <div className="space-y-4">
            {/* Summary */}
            <div className={`p-4 rounded-lg ${verifyResult.valid ? 'bg-green-50 border border-green-200' : 'bg-red-50 border border-red-200'}`}>
              <div className="flex items-center gap-2 mb-2">
                {verifyResult.valid ? (
                  <CheckCircle className="h-5 w-5 text-green-600" />
                ) : (
                  <XCircle className="h-5 w-5 text-red-600" />
                )}
                <span className={`font-medium ${verifyResult.valid ? 'text-green-800' : 'text-red-800'}`}>
                  {verifyResult.valid ? 'Lanac je validan' : 'Detektovani problemi!'}
                </span>
              </div>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                <div>
                  <div className="text-gray-500">Provereno</div>
                  <div className="font-mono font-medium">{verifyResult.checked}</div>
                </div>
                <div>
                  <div className="text-gray-500">Content valid</div>
                  <div className="font-mono font-medium text-green-600">{verifyResult.content_valid}</div>
                </div>
                <div>
                  <div className="text-gray-500">Content invalid</div>
                  <div className="font-mono font-medium text-red-600">{verifyResult.content_invalid}</div>
                </div>
                <div>
                  <div className="text-gray-500">Linkage valid</div>
                  <div className="font-mono font-medium text-green-600">{verifyResult.linkage_valid}</div>
                </div>
              </div>
            </div>

            {/* Violations */}
            {verifyResult.violations && verifyResult.violations.length > 0 && (
              <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                <div className="font-medium text-red-800 mb-2">Pronađene nepravilnosti:</div>
                <ul className="text-sm text-red-700 space-y-1">
                  {verifyResult.violations.map((v, i) => (
                    <li key={i} className="font-mono">{v}</li>
                  ))}
                </ul>
              </div>
            )}

            {/* Entry details */}
            {verifyResult.entries && verifyResult.entries.length > 0 && (
              <div>
                <div className="text-sm font-medium text-gray-700 mb-2">Detalji po zapisima:</div>
                <div className="max-h-64 overflow-y-auto border border-gray-200 rounded-lg">
                  <table className="min-w-full divide-y divide-gray-200 text-xs">
                    <thead className="bg-gray-50 sticky top-0">
                      <tr>
                        <th className="px-3 py-2 text-left font-medium text-gray-500">Seq</th>
                        <th className="px-3 py-2 text-left font-medium text-gray-500">Action</th>
                        <th className="px-3 py-2 text-left font-medium text-gray-500">Content</th>
                        <th className="px-3 py-2 text-left font-medium text-gray-500">Linkage</th>
                        <th className="px-3 py-2 text-left font-medium text-gray-500">Hash</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200">
                      {verifyResult.entries.map((entry) => (
                        <tr key={entry.id} className={entry.valid ? '' : 'bg-red-50'}>
                          <td className="px-3 py-2 font-mono">{entry.sequence}</td>
                          <td className="px-3 py-2">{entry.action}</td>
                          <td className="px-3 py-2">
                            {entry.content_valid ? (
                              <CheckCircle className="h-4 w-4 text-green-500" />
                            ) : (
                              <XCircle className="h-4 w-4 text-red-500" />
                            )}
                          </td>
                          <td className="px-3 py-2">
                            {entry.linkage_valid ? (
                              <CheckCircle className="h-4 w-4 text-green-500" />
                            ) : (
                              <XCircle className="h-4 w-4 text-red-500" />
                            )}
                          </td>
                          <td className="px-3 py-2 font-mono text-gray-500">{entry.hash.substring(0, 12)}...</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Checkpoints */}
      <div className="border border-gray-200 rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="font-medium text-gray-900">Checkpointi</h3>
            <p className="text-sm text-gray-500">Snimi trenutno stanje lanca sa internim svedokom</p>
          </div>
          <button
            onClick={handleCreateCheckpoint}
            disabled={isCreatingCheckpoint}
            className="flex items-center gap-2 px-4 py-2 bg-accent-600 text-white rounded-lg hover:bg-accent-700 disabled:opacity-50"
          >
            {isCreatingCheckpoint ? (
              <RefreshCw className="h-4 w-4 animate-spin" />
            ) : (
              <Database className="h-4 w-4" />
            )}
            Kreiraj checkpoint
          </button>
        </div>

        {checkpoints?.data && checkpoints.data.length > 0 ? (
          <div className="space-y-3">
            {checkpoints.data.map((cp) => (
              <div key={cp.id} className="bg-gray-50 rounded-lg p-4">
                <div className="flex items-center justify-between mb-2">
                  <div className="font-mono text-sm text-gray-700">
                    Seq #{cp.last_sequence} • {cp.entry_count} entries
                  </div>
                  <span className={`text-xs px-2 py-1 rounded ${
                    cp.witness_status === 'confirmed'
                      ? 'bg-green-100 text-green-700'
                      : 'bg-yellow-100 text-yellow-700'
                  }`}>
                    {cp.witness_type} - {cp.witness_status}
                  </span>
                </div>
                <div className="font-mono text-xs text-gray-500 truncate">
                  Hash: {cp.checkpoint_hash}
                </div>
                <div className="text-xs text-gray-400 mt-1">
                  {new Date(cp.created_at).toLocaleString('sr-RS')}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center text-gray-500 py-8">
            Nema checkpointa. Kreirajte prvi checkpoint da testirate sistem.
          </div>
        )}
      </div>
    </div>
  )
}

function CodeBlock({
  title,
  code,
  language,
  expanded,
  onToggle,
}: {
  title: string
  code: string
  language: string
  expanded: boolean
  onToggle: () => void
}) {
  return (
    <div className="border border-gray-200 rounded-lg overflow-hidden">
      <button
        onClick={onToggle}
        className="w-full flex items-center justify-between px-4 py-3 bg-gray-50 hover:bg-gray-100 transition-colors"
      >
        <div className="flex items-center gap-2">
          <FileCode className="h-4 w-4 text-gray-500" />
          <span className="font-medium text-gray-700">{title}</span>
          <span className="text-xs text-gray-400 px-2 py-0.5 bg-gray-200 rounded">{language}</span>
        </div>
        {expanded ? (
          <ChevronDown className="h-4 w-4 text-gray-500" />
        ) : (
          <ChevronRight className="h-4 w-4 text-gray-500" />
        )}
      </button>
      {expanded && (
        <div className="bg-gray-900 p-4 overflow-x-auto">
          <pre className="text-sm text-gray-100 font-mono whitespace-pre">{code}</pre>
        </div>
      )}
    </div>
  )
}
