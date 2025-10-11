// Script para inicializar proveedores de publicidad
// Ejecutar con: node init-providers.js [DB_URL]

const { MongoClient } = require('mongodb');
const fs = require('fs');
const path = require('path');

async function initProviders() {
    // Intentar obtener DB_URL
    let uri = process.argv[2];
    
    if (!uri) {
        try {
            const envPath = path.join(__dirname, '..', '.env');
            const envContent = fs.readFileSync(envPath, 'utf8');
            const dbUrlMatch = envContent.match(/DB_URL=(.+)/);
            const mongoUrlMatch = envContent.match(/MONGO_URL=(.+)/);
            uri = (dbUrlMatch && dbUrlMatch[1]) || (mongoUrlMatch && mongoUrlMatch[1]);
        } catch (error) {
            // Ignorar error
        }
    }
    
    if (!uri) {
        uri = process.env.DB_URL || process.env.MONGO_URL;
    }

    if (!uri) {
        console.error('❌ Error: No se encontró DB_URL');
        console.log('\n💡 Ejecuta el script de una de estas formas:');
        console.log('   1. node init-providers.js "mongodb+srv://user:pass@cluster.mongodb.net/opo"');
        console.log('   2. Configura DB_URL en el archivo .env');
        process.exit(1);
    }
    
    const dbName = process.env.DB_NAME || 'opo';

    console.log('🔗 Conectando a MongoDB...');
    console.log(`📦 Base de datos: ${dbName}`);
    
    const client = new MongoClient(uri);

    try {
        await client.connect();
        console.log('✅ Conectado a MongoDB');

        const database = client.db(dbName);
        const collection = database.collection('ad_providers');

        // Verificar si ya existen proveedores
        const count = await collection.countDocuments({});
        console.log(`\n📊 Proveedores existentes: ${count}`);

        if (count > 0) {
            console.log('⚠️  Ya existen proveedores en la base de datos.');
            console.log('   Para reinicializar, primero elimina los proveedores existentes.');
            return;
        }

        // Proveedores iniciales
        const providers = [
            {
                _id: generateUUID(),
                providerId: 'admob',
                name: 'AdMob',
                icon: 'ads_click',
                color: '#4285f4',
                enabled: true,
                order: 1,
                createdAt: new Date(),
                updatedAt: new Date()
            },
            {
                _id: generateUUID(),
                providerId: 'facebook',
                name: 'Facebook Audience Network',
                icon: 'campaign',
                color: '#1877f2',
                enabled: true,
                order: 2,
                createdAt: new Date(),
                updatedAt: new Date()
            },
            {
                _id: generateUUID(),
                providerId: 'unity',
                name: 'Unity Ads',
                icon: 'videogame_asset',
                color: '#000000',
                enabled: true,
                order: 3,
                createdAt: new Date(),
                updatedAt: new Date()
            },
            {
                _id: generateUUID(),
                providerId: 'custom',
                name: 'Personalizado',
                icon: 'settings',
                color: '#757575',
                enabled: true,
                order: 99,
                createdAt: new Date(),
                updatedAt: new Date()
            }
        ];

        const result = await collection.insertMany(providers);

        console.log(`\n✅ Proveedores inicializados exitosamente:`);
        console.log(`   - Documentos insertados: ${result.insertedCount}`);
        
        console.log(`\n📋 Proveedores creados:`);
        providers.forEach((provider, index) => {
            console.log(`   ${index + 1}. ${provider.name} (${provider.providerId}) - ${provider.enabled ? 'Habilitado' : 'Deshabilitado'}`);
        });

    } catch (error) {
        console.error('❌ Error:', error);
        process.exit(1);
    } finally {
        await client.close();
        console.log('\n🔌 Conexión cerrada');
    }
}

function generateUUID() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
        const r = Math.random() * 16 | 0;
        const v = c == 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
}

initProviders();

