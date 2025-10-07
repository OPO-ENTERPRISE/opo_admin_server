// Script para agregar el campo 'enabled' a todos los usuarios
// Ejecutar con: node add-enabled-users.js [DB_URL]
// Ejemplo: node add-enabled-users.js "mongodb+srv://user:pass@cluster.mongodb.net/opo"

const { MongoClient } = require('mongodb');
const fs = require('fs');
const path = require('path');

async function addEnabledField() {
    // Intentar obtener DB_URL de múltiples fuentes
    let uri = process.argv[2]; // Desde argumento de línea de comandos
    
    if (!uri) {
        // Intentar leer del archivo .env
        try {
            const envPath = path.join(__dirname, '..', '.env');
            const envContent = fs.readFileSync(envPath, 'utf8');
            const dbUrlMatch = envContent.match(/DB_URL=(.+)/);
            const mongoUrlMatch = envContent.match(/MONGO_URL=(.+)/);
            uri = (dbUrlMatch && dbUrlMatch[1]) || (mongoUrlMatch && mongoUrlMatch[1]);
        } catch (error) {
            // Archivo .env no encontrado o error leyéndolo
        }
    }
    
    if (!uri) {
        uri = process.env.DB_URL || process.env.MONGO_URL;
    }

    if (!uri) {
        console.error('❌ Error: No se encontró DB_URL');
        console.log('\n💡 Ejecuta el script de una de estas formas:');
        console.log('   1. node add-enabled-users.js "mongodb+srv://user:pass@cluster.mongodb.net/opo"');
        console.log('   2. Configura DB_URL en el archivo .env');
        console.log('   3. Configura la variable de entorno DB_URL');
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
        const collection = database.collection('users');

        // Contar documentos sin el campo enabled
        const countWithoutEnabled = await collection.countDocuments({ 
            enabled: { $exists: false } 
        });
        console.log(`\n📊 Usuarios sin campo 'enabled': ${countWithoutEnabled}`);

        if (countWithoutEnabled === 0) {
            console.log('✅ Todos los usuarios ya tienen el campo enabled definido');
            return;
        }

        // Actualizar todos los documentos que no tienen el campo enabled
        const result = await collection.updateMany(
            { enabled: { $exists: false } },
            { $set: { enabled: false } }
        );

        console.log(`\n✅ Operación completada:`);
        console.log(`   - Documentos encontrados: ${result.matchedCount}`);
        console.log(`   - Documentos actualizados: ${result.modifiedCount}`);

        // Verificar el resultado
        const totalUsers = await collection.countDocuments({});
        const enabledUsers = await collection.countDocuments({ enabled: true });
        const disabledUsers = await collection.countDocuments({ enabled: false });

        console.log(`\n📊 Estadísticas finales:`);
        console.log(`   - Total de usuarios: ${totalUsers}`);
        console.log(`   - Usuarios habilitados (enabled: true): ${enabledUsers}`);
        console.log(`   - Usuarios deshabilitados (enabled: false): ${disabledUsers}`);

        // Mostrar algunos ejemplos
        console.log(`\n🔍 Ejemplos de usuarios actualizados:`);
        const samples = await collection.find({ enabled: false }).limit(3).toArray();
        samples.forEach((user, index) => {
            console.log(`   ${index + 1}. Name: ${user.name}, Email: ${user.email}, Area: ${user.area}, Enabled: ${user.enabled}`);
        });

    } catch (error) {
        console.error('❌ Error:', error);
        process.exit(1);
    } finally {
        await client.close();
        console.log('\n🔌 Conexión cerrada');
    }
}

addEnabledField();

