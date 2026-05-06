const mongoHost = process.env.INVENTORY_API_MONGODB_HOST
const mongoPort = process.env.INVENTORY_API_MONGODB_PORT

const mongoUser = process.env.INVENTORY_API_MONGODB_USERNAME
const mongoPassword = process.env.INVENTORY_API_MONGODB_PASSWORD

const database = process.env.INVENTORY_API_MONGODB_DATABASE

const retrySeconds = parseInt(process.env.RETRY_CONNECTION_SECONDS || "5") || 5;

const collections = {
    equipment: "equipment",
    locations: "locations",
    serviceRequests: "service_requests",
};

// try to connect to mongoDB until it is not available
let connection;
while(true) {
    try {
        connection = Mongo(`mongodb://${mongoUser}:${mongoPassword}@${mongoHost}:${mongoPort}`);
        break;
    } catch (exception) {
        print(`Cannot connect to mongoDB: ${exception}`);
        print(`Will retry after ${retrySeconds} seconds`)
        sleep(retrySeconds * 1000);
    }
}

const db = connection.getDB(database)
const existingCollections = db.getCollectionNames()

if (!existingCollections.includes(collections.locations)) {
    print(`Creating collection '${collections.locations}'`)
    db.createCollection(collections.locations)
    db[collections.locations].createIndex({ "id": 1 }, { unique: true })

    db[collections.locations].insertMany([
        {
            "id": "loc-a101",
            "building": "A",
            "floor": "1",
            "department": "Radiology",
            "room": "A101",
            "description": "Main radiology room",
            "equipmentCount": 0,
            "createdAt": new Date(),
            "updatedAt": new Date()
        },
        {
            "id": "loc-b202",
            "building": "B",
            "floor": "2",
            "department": "Surgery",
            "room": "B202",
            "description": "Operating theatre 2",
            "equipmentCount": 0,
            "createdAt": new Date(),
            "updatedAt": new Date()
        }
    ])
} else {
    print(`Collection '${collections.locations}' already exists, skipping`)
}

if (!existingCollections.includes(collections.equipment)) {
    print(`Creating collection '${collections.equipment}'`)
    db.createCollection(collections.equipment)
    db[collections.equipment].createIndex({ "id": 1 }, { unique: true })
    db[collections.equipment].createIndex({ "locationId": 1 })
    db[collections.equipment].createIndex({ "status": 1 })

    db[collections.equipment].insertMany([
        {
            "id": "eq-xray-001",
            "name": "X-Ray Machine",
            "type": "Imaging",
            "inventoryNumber": "INV-2024-001",
            "serialNumber": "XR-9900-A",
            "manufacturer": "Siemens",
            "model": "YSIO Max",
            "purchaseDate": "2022-03-15",
            "warrantyExpiry": "2027-03-15",
            "lifespanYears": 10,
            "locationId": "loc-a101",
            "status": "ACTIVE",
            "openServiceRequestCount": 0,
            "createdAt": new Date(),
            "updatedAt": new Date()
        },
        {
            "id": "eq-monitor-002",
            "name": "Patient Monitor",
            "type": "Monitoring",
            "inventoryNumber": "INV-2024-002",
            "serialNumber": "PM-4412-B",
            "manufacturer": "Philips",
            "model": "IntelliVue MX450",
            "purchaseDate": "2023-06-01",
            "warrantyExpiry": "2026-06-01",
            "lifespanYears": 7,
            "locationId": "loc-b202",
            "status": "ACTIVE",
            "openServiceRequestCount": 1,
            "createdAt": new Date(),
            "updatedAt": new Date()
        }
    ])
} else {
    print(`Collection '${collections.equipment}' already exists, skipping`)
}

if (!existingCollections.includes(collections.serviceRequests)) {
    print(`Creating collection '${collections.serviceRequests}'`)
    db.createCollection(collections.serviceRequests)
    db[collections.serviceRequests].createIndex({ "id": 1 }, { unique: true })
    db[collections.serviceRequests].createIndex({ "equipmentId": 1 })
    db[collections.serviceRequests].createIndex({ "status": 1 })

    db[collections.serviceRequests].insertMany([
        {
            "id": "sr-001",
            "title": "Monitor display flickering",
            "description": "The patient monitor in B202 shows intermittent display flickering during use.",
            "priority": "MEDIUM",
            "equipmentId": "eq-monitor-002",
            "reportedBy": "nurse.jana",
            "status": "NEW",
            "createdAt": new Date(),
            "updatedAt": new Date()
        }
    ])
} else {
    print(`Collection '${collections.serviceRequests}' already exists, skipping`)
}

print("Database initialization complete")
process.exit(0);
