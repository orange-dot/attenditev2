export type CityId = 'kikinda' | 'beograd' | 'kragujevac'
export type InstitutionType = 'local' | 'central' | 'datacenter'

export interface Institution {
  id: string
  name: string
  shortName: string
  city: CityId
  coords: [number, number]
  type: InstitutionType
  description: string
}

export interface FlowStep {
  id: string
  from: string
  to: string
  procedural: string
  technical: string
  dataExchanged: string[]
  duration: number
  isResponse?: boolean
}

export interface UseCase {
  id: string
  title: string
  subtitle: string
  description: string
  traditionalTime: string
  newTime: string
  institutions: string[]
  steps: FlowStep[]
}

export const cities: Record<CityId, { name: string; coords: [number, number] }> = {
  kikinda: { name: 'Kikinda', coords: [45.8297, 20.4656] },
  beograd: { name: 'Beograd', coords: [44.8125, 20.4612] },
  kragujevac: { name: 'Kragujevac', coords: [44.0128, 20.9114] },
}

export const institutions: Institution[] = [
  // Kikinda - lokalne institucije
  {
    id: 'geronto-kikinda',
    name: 'Gerontološki centar Kikinda',
    shortName: 'Gerontološki',
    city: 'kikinda',
    coords: [45.8297, 20.4656],
    type: 'local',
    description: 'Ustanova za smeštaj i negu starijih osoba',
  },
  {
    id: 'csr-kikinda',
    name: 'Centar za socijalni rad Kikinda',
    shortName: 'CSR',
    city: 'kikinda',
    coords: [45.8265, 20.4590],
    type: 'local',
    description: 'Lokalni centar za socijalni rad',
  },
  {
    id: 'mup-kikinda',
    name: 'Policijska stanica Kikinda',
    shortName: 'MUP Kikinda',
    city: 'kikinda',
    coords: [45.8320, 20.4700],
    type: 'local',
    description: 'Lokalna policijska stanica',
  },

  // Beograd - centralne institucije
  {
    id: 'mup-srbije',
    name: 'MUP Srbije - Centralni registar',
    shortName: 'MUP Centralni',
    city: 'beograd',
    coords: [44.8125, 20.4612],
    type: 'central',
    description: 'Centralni registar građana, lična dokumenta, prebivalište',
  },
  {
    id: 'poreska',
    name: 'Poreska uprava Srbije',
    shortName: 'Poreska',
    city: 'beograd',
    coords: [44.8180, 20.4550],
    type: 'central',
    description: 'Evidencija prihoda, poreza i zaposlenja',
  },
  {
    id: 'katastar',
    name: 'Republički geodetski zavod',
    shortName: 'RGZ Katastar',
    city: 'beograd',
    coords: [44.8050, 20.4700],
    type: 'central',
    description: 'Evidencija nepokretnosti i vlasništva',
  },

  // Kragujevac - Data centar
  {
    id: 'data-centar',
    name: 'Data centar Vlade Srbije',
    shortName: 'Data centar',
    city: 'kragujevac',
    coords: [44.0128, 20.9114],
    type: 'datacenter',
    description: 'Centralni hub za bezbednu razmenu podataka između institucija',
  },
]

export const useCases: UseCase[] = [
  {
    id: 'gerontoloski-smestaj',
    title: 'Smeštaj u gerontološki centar',
    subtitle: 'Prijava starijeg člana porodice za institucionalni smeštaj',
    description:
      'Srodnik želi da smesti starijeg člana porodice u gerontološki centar. Tradicionalno, to zahteva prikupljanje dokumenata iz više institucija. Sa novim sistemom, sve se završava na jednom šalteru.',
    traditionalTime: '2-3 nedelje',
    newTime: '1-2 dana',
    institutions: ['geronto-kikinda', 'data-centar', 'mup-srbije', 'poreska', 'csr-kikinda'],
    steps: [
      {
        id: 'step-1',
        from: 'citizen',
        to: 'geronto-kikinda',
        procedural: 'Srodnik dolazi u Gerontološki centar sa ličnom kartom i JMBG korisnika',
        technical: 'Službenik unosi JMBG u sistem i pokreće zahtev za verifikaciju',
        dataExchanged: ['JMBG korisnika', 'JMBG srodnika'],
        duration: 1500,
      },
      {
        id: 'step-2',
        from: 'geronto-kikinda',
        to: 'data-centar',
        procedural: 'Srodnik čeka dok sistem obrađuje zahtev',
        technical: 'Gerontološki centar šalje zahtev ka Data centru',
        dataExchanged: ['Zahtev za verifikaciju', 'Tip usluge: smeštaj'],
        duration: 1500,
      },
      {
        id: 'step-3',
        from: 'data-centar',
        to: 'mup-srbije',
        procedural: 'Sistem automatski proverava podatke',
        technical: 'Data centar šalje upit MUP-u za verifikaciju identiteta i srodstva',
        dataExchanged: ['Upit: identitet', 'Upit: srodstvo', 'Upit: adresa'],
        duration: 1500,
      },
      {
        id: 'step-4',
        from: 'mup-srbije',
        to: 'data-centar',
        procedural: '',
        technical: 'MUP vraća potvrdu identiteta i srodstva (samo DA/NE + osnovni podaci)',
        dataExchanged: ['Potvrda identiteta: DA', 'Srodstvo: sin/ćerka', 'Adresa: potvrđena'],
        duration: 1500,
        isResponse: true,
      },
      {
        id: 'step-5',
        from: 'data-centar',
        to: 'poreska',
        procedural: '',
        technical: 'Data centar šalje upit Poreskoj za proveru prihoda',
        dataExchanged: ['Upit: prihodi korisnika', 'Upit: penzija'],
        duration: 1500,
      },
      {
        id: 'step-6',
        from: 'poreska',
        to: 'data-centar',
        procedural: '',
        technical: 'Poreska vraća kategorizaciju prihoda (bez tačnih iznosa)',
        dataExchanged: ['Kategorija prihoda: niska/srednja/visoka', 'Penzioner: DA'],
        duration: 1500,
        isResponse: true,
      },
      {
        id: 'step-7',
        from: 'data-centar',
        to: 'csr-kikinda',
        procedural: '',
        technical: 'Data centar proverava postojeće usluge u CSR',
        dataExchanged: ['Upit: aktivne usluge', 'Upit: prethodne prijave'],
        duration: 1500,
      },
      {
        id: 'step-8',
        from: 'csr-kikinda',
        to: 'data-centar',
        procedural: '',
        technical: 'CSR vraća status (nema aktivnih usluga)',
        dataExchanged: ['Aktivne usluge: 0', 'Prethodne prijave: nema'],
        duration: 1500,
        isResponse: true,
      },
      {
        id: 'step-9',
        from: 'data-centar',
        to: 'geronto-kikinda',
        procedural: 'Službenik dobija obaveštenje da je verifikacija završena',
        technical: 'Data centar šalje agregiran odgovor Gerontološkom centru',
        dataExchanged: ['Status: ODOBREN', 'Sve provere: uspešne', 'Kategorija: prioritetna'],
        duration: 1500,
        isResponse: true,
      },
      {
        id: 'step-10',
        from: 'geronto-kikinda',
        to: 'citizen',
        procedural: 'Srodnik dobija potvrdu o prijemu zahteva i procenjeni rok',
        technical: 'Sistem generiše potvrdu sa jedinstvenim brojem predmeta',
        dataExchanged: ['Potvrda prijema', 'Broj predmeta', 'Procenjeni rok'],
        duration: 1500,
        isResponse: true,
      },
    ],
  },
  {
    id: 'socijalna-pomoc',
    title: 'Zahtev za socijalnu pomoć',
    subtitle: 'Prijava za novčanu socijalnu pomoć',
    description:
      'Građanin podnosi zahtev za novčanu socijalnu pomoć. Tradicionalno mora da prikupi uverenja iz MUP-a, Poreske i Katastra. Sa novim sistemom, dovoljno je da dođe u CSR sa ličnom kartom.',
    traditionalTime: '2-4 nedelje',
    newTime: '2-3 dana',
    institutions: ['csr-kikinda', 'data-centar', 'mup-srbije', 'poreska', 'katastar'],
    steps: [
      {
        id: 'step-1',
        from: 'citizen',
        to: 'csr-kikinda',
        procedural: 'Građanin dolazi u CSR Kikinda sa ličnom kartom',
        technical: 'Službenik unosi JMBG i pokreće zahtev za socijalnu pomoć',
        dataExchanged: ['JMBG podnosioca', 'Tip zahteva: NSP'],
        duration: 1500,
      },
      {
        id: 'step-2',
        from: 'csr-kikinda',
        to: 'data-centar',
        procedural: 'Građanin čeka dok sistem proverava uslove',
        technical: 'CSR šalje zahtev za proveru uslova ka Data centru',
        dataExchanged: ['Zahtev za verifikaciju', 'Tip: novčana socijalna pomoć'],
        duration: 1500,
      },
      {
        id: 'step-3',
        from: 'data-centar',
        to: 'mup-srbije',
        procedural: '',
        technical: 'Data centar šalje upit MUP-u za identitet i domaćinstvo',
        dataExchanged: ['Upit: identitet', 'Upit: članovi domaćinstva', 'Upit: prebivalište'],
        duration: 1500,
      },
      {
        id: 'step-4',
        from: 'mup-srbije',
        to: 'data-centar',
        procedural: '',
        technical: 'MUP vraća podatke o domaćinstvu',
        dataExchanged: ['Broj članova: 3', 'Prebivalište: potvrđeno', 'Državljanstvo: DA'],
        duration: 1500,
        isResponse: true,
      },
      {
        id: 'step-5',
        from: 'data-centar',
        to: 'poreska',
        procedural: '',
        technical: 'Data centar proverava prihode svih članova domaćinstva',
        dataExchanged: ['Upit: prihodi', 'Upit: zaposlenje', 'Za sve članove domaćinstva'],
        duration: 1500,
      },
      {
        id: 'step-6',
        from: 'poreska',
        to: 'data-centar',
        procedural: '',
        technical: 'Poreska vraća sumarni prihod domaćinstva',
        dataExchanged: ['Ukupni prihod: ispod cenzusa', 'Zaposleni članovi: 0'],
        duration: 1500,
        isResponse: true,
      },
      {
        id: 'step-7',
        from: 'data-centar',
        to: 'katastar',
        procedural: '',
        technical: 'Data centar proverava imovinu u Katastru',
        dataExchanged: ['Upit: nepokretnosti', 'Za sve članove domaćinstva'],
        duration: 1500,
      },
      {
        id: 'step-8',
        from: 'katastar',
        to: 'data-centar',
        procedural: '',
        technical: 'Katastar vraća status imovine',
        dataExchanged: ['Nepokretnosti: 1 stan (ispod cenzusa)', 'Dodatna imovina: nema'],
        duration: 1500,
        isResponse: true,
      },
      {
        id: 'step-9',
        from: 'data-centar',
        to: 'csr-kikinda',
        procedural: 'Službenik dobija rezultat provere',
        technical: 'Data centar šalje agregiran rezultat CSR-u',
        dataExchanged: ['Status: ISPUNJAVA USLOVE', 'Kategorija: redovna pomoć', 'Preporučeni iznos'],
        duration: 1500,
        isResponse: true,
      },
      {
        id: 'step-10',
        from: 'csr-kikinda',
        to: 'citizen',
        procedural: 'Građanin dobija informaciju o statusu zahteva i daljem postupku',
        technical: 'Sistem kreira predmet i obaveštava građanina',
        dataExchanged: ['Potvrda prijema', 'Broj predmeta', 'Rok za rešenje'],
        duration: 1500,
        isResponse: true,
      },
    ],
  },
]

export function getInstitutionById(id: string): Institution | undefined {
  return institutions.find((i) => i.id === id)
}

export function getInstitutionsForUseCase(useCase: UseCase): Institution[] {
  return useCase.institutions
    .map((id) => getInstitutionById(id))
    .filter((i): i is Institution => i !== undefined)
}
