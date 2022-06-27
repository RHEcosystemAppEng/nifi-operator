package nifi

import (
	"context"
	"reflect"

	bigdatav1alpha1 "github.com/RHEcosystemAppEng/nifi-operator/api/v1alpha1"
	nifiutils "github.com/RHEcosystemAppEng/nifi-operator/controllers/nifiutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// nifiPropertiesConfigMapName
	nifiPropertiesConfigMapNameSuffix = "-nifi-properties"
)

func newConfigMap(nifi *bigdatav1alpha1.Nifi) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifi.Name,
			Namespace: nifi.Namespace,
			Labels:    labelsForNifi(nifi.Name),
		},
	}
}

func newConfigMapWithName(name string, nifi *bigdatav1alpha1.Nifi) *corev1.ConfigMap {
	cm := newConfigMap(nifi)
	cm.ObjectMeta.Name = name
	return cm
}

func getDefaultNifiProperties() *map[string]string {
	nifiProperties := make(map[string]string)
	nifiProperties["nifi.flow.configuration.file"] = "./conf/flow.xml.gz"
	nifiProperties["nifi.flow.configuration.json.file"] = "./conf/flow.json.gz"
	nifiProperties["nifi.flow.configuration.archive.enabled"] = "true"
	nifiProperties["nifi.flow.configuration.archive.dir"] = "./conf/archive/"
	nifiProperties["nifi.flow.configuration.archive.max.time"] = "30 days"
	nifiProperties["nifi.flow.configuration.archive.max.storage"] = "500 MB"
	nifiProperties["nifi.flow.configuration.archive.max.count"] = ""
	nifiProperties["nifi.flowcontroller.autoResumeState"] = "true"
	nifiProperties["nifi.flowcontroller.graceful.shutdown.period"] = "10 sec"
	nifiProperties["nifi.flowservice.writedelay.interval"] = "500 ms"
	nifiProperties["nifi.administrative.yield.duration"] = "30 sec"
	nifiProperties["nifi.bored.yield.duration"] = "10 millis"
	nifiProperties["nifi.queue.backpressure.count"] = "10000"
	nifiProperties["nifi.queue.backpressure.size"] = "1 GB"
	nifiProperties["nifi.authorizer.configuration.file"] = "./conf/authorizers.xml"
	nifiProperties["nifi.login.identity.provider.configuration.file"] = "./conf/login-identity-providers.xml"
	nifiProperties["nifi.templates.directory"] = "./conf/templates"
	nifiProperties["nifi.ui.banner.text"] = ""
	nifiProperties["nifi.ui.autorefresh.interval"] = "30 sec"
	nifiProperties["nifi.nar.library.directory"] = "./lib"
	nifiProperties["nifi.nar.library.autoload.directory"] = "./extensions"
	nifiProperties["nifi.nar.working.directory"] = "./work/nar/"
	nifiProperties["nifi.documentation.working.directory"] = "./work/docs/components"
	nifiProperties["nifi.state.management.configuration.file"] = "./conf/state-management.xml"
	nifiProperties["nifi.state.management.provider.local"] = "local-provider"
	nifiProperties["nifi.state.management.provider.cluster"] = "zk-provider"
	nifiProperties["nifi.state.management.embedded.zookeeper.start"] = "false"
	nifiProperties["nifi.state.management.embedded.zookeeper.properties"] = "./conf/zookeeper.properties"
	nifiProperties["nifi.database.directory"] = "./database_repository"
	nifiProperties["nifi.h2.url.append"] = ";LOCK_TIMEOUT=25000;WRITE_DELAY=0;AUTO_SERVER=FALSE"
	nifiProperties["nifi.repository.encryption.protocol.version"] = ""
	nifiProperties["nifi.repository.encryption.key.id"] = ""
	nifiProperties["nifi.repository.encryption.key.provider"] = ""
	nifiProperties["nifi.repository.encryption.key.provider.keystore.location"] = ""
	nifiProperties["nifi.repository.encryption.key.provider.keystore.password"] = ""
	nifiProperties["nifi.flowfile.repository.implementation"] = "org.apache.nifi.controller.repository.WriteAheadFlowFileRepository"
	nifiProperties["nifi.flowfile.repository.wal.implementation"] = "org.apache.nifi.wali.SequentialAccessWriteAheadLog"
	nifiProperties["nifi.flowfile.repository.directory"] = "./flowfile_repository"
	nifiProperties["nifi.flowfile.repository.checkpoint.interval"] = "20 secs"
	nifiProperties["nifi.flowfile.repository.always.sync"] = "false"
	nifiProperties["nifi.flowfile.repository.retain.orphaned.flowfiles"] = "true"
	nifiProperties["nifi.swap.manager.implementation"] = "org.apache.nifi.controller.FileSystemSwapManager"
	nifiProperties["nifi.queue.swap.threshold"] = "20000"
	nifiProperties["nifi.content.repository.implementation"] = "org.apache.nifi.controller.repository.FileSystemRepository"
	nifiProperties["nifi.content.claim.max.appendable.size"] = "50 KB"
	nifiProperties["nifi.content.repository.directory.default"] = "./content_repository"
	nifiProperties["nifi.content.repository.archive.max.retention.period"] = "7 days"
	nifiProperties["nifi.content.repository.archive.max.usage.percentage"] = "50%"
	nifiProperties["nifi.content.repository.archive.enabled"] = "true"
	nifiProperties["nifi.content.repository.always.sync"] = "false"
	nifiProperties["nifi.content.viewer.url"] = "../nifi-content-viewer/"
	nifiProperties["nifi.provenance.repository.implementation"] = "org.apache.nifi.provenance.WriteAheadProvenanceRepository"
	nifiProperties["nifi.provenance.repository.directory.default"] = "./provenance_repository"
	nifiProperties["nifi.provenance.repository.max.storage.time"] = "30 days"
	nifiProperties["nifi.provenance.repository.max.storage.size"] = "10 GB"
	nifiProperties["nifi.provenance.repository.rollover.time"] = "10 mins"
	nifiProperties["nifi.provenance.repository.rollover.size"] = "100 MB"
	nifiProperties["nifi.provenance.repository.query.threads"] = "2"
	nifiProperties["nifi.provenance.repository.index.threads"] = "2"
	nifiProperties["nifi.provenance.repository.compress.on.rollover"] = "true"
	nifiProperties["nifi.provenance.repository.always.sync"] = "false"
	nifiProperties["nifi.provenance.repository.indexed.fields"] = "EventType, FlowFileUUID, Filename, ProcessorID, Relationship"
	nifiProperties["nifi.provenance.repository.indexed.attributes"] = ""
	nifiProperties["nifi.provenance.repository.index.shard.size"] = "500 MB"
	nifiProperties["nifi.provenance.repository.max.attribute.length"] = "65536"
	nifiProperties["nifi.provenance.repository.concurrent.merge.threads"] = "2"
	nifiProperties["nifi.provenance.repository.buffer.size"] = "100000"
	nifiProperties["nifi.components.status.repository.implementation"] = "org.apache.nifi.controller.status.history.VolatileComponentStatusRepository"
	nifiProperties["nifi.components.status.repository.buffer.size"] = "1440"
	nifiProperties["nifi.components.status.snapshot.frequency"] = "1 min"
	nifiProperties["nifi.status.repository.questdb.persist.node.days"] = "14"
	nifiProperties["nifi.status.repository.questdb.persist.component.days"] = "3"
	nifiProperties["nifi.status.repository.questdb.persist.location"] = "./status_repository"
	nifiProperties["nifi.remote.input.host"] = "nifi-sample-0"
	nifiProperties["nifi.remote.input.secure"] = "true"
	nifiProperties["nifi.remote.input.socket.port"] = "10000"
	nifiProperties["nifi.remote.input.http.enabled"] = "true"
	nifiProperties["nifi.remote.input.http.transaction.ttl"] = "30 sec"
	nifiProperties["nifi.remote.contents.cache.expiration"] = "30 secs"
	nifiProperties["nifi.web.http.host"] = ""
	nifiProperties["nifi.web.http.port"] = ""
	nifiProperties["nifi.web.http.network.interface.default"] = ""
	nifiProperties["nifi.web.https.host"] = "0.0.0.0"
	nifiProperties["nifi.web.https.port"] = "8443"
	nifiProperties["nifi.web.https.network.interface.default"] = ""
	nifiProperties["nifi.web.jetty.working.directory"] = "./work/jetty"
	nifiProperties["nifi.web.jetty.threads"] = "200"
	nifiProperties["nifi.web.max.header.size"] = "16 KB"
	nifiProperties["nifi.web.proxy.context.path"] = ""
	nifiProperties["nifi.web.proxy.host"] = ""
	nifiProperties["nifi.web.max.content.size"] = ""
	nifiProperties["nifi.web.max.requests.per.second"] = "30000"
	nifiProperties["nifi.web.max.access.token.requests.per.second"] = "25"
	nifiProperties["nifi.web.request.timeout"] = "60 secs"
	nifiProperties["nifi.web.request.ip.whitelist"] = ""
	nifiProperties["nifi.web.should.send.server.version"] = "true"
	nifiProperties["nifi.web.request.log.format"] = "%{client}a - %u %t \"%r\" %s %O \"%{Referer}i\" \"%{User-Agent}i\""
	nifiProperties["nifi.web.https.ciphersuites.include"] = ""
	nifiProperties["nifi.web.https.ciphersuites.exclude"] = ""
	nifiProperties["nifi.sensitive.props.key"] = "TzzT/ZbzY1vIsM3T3caxvNpjJmNeQRL5"
	nifiProperties["nifi.sensitive.props.key.protected"] = ""
	nifiProperties["nifi.sensitive.props.algorithm"] = "NIFI_PBKDF2_AES_GCM_256"
	nifiProperties["nifi.sensitive.props.additional.keys"] = ""
	nifiProperties["nifi.security.autoreload.enabled"] = "false"
	nifiProperties["nifi.security.autoreload.interval"] = "10 secs"
	nifiProperties["nifi.security.keystore"] = "./conf/keystore.p12"
	nifiProperties["nifi.security.keystoreType"] = "PKCS12"
	nifiProperties["nifi.security.keystorePasswd"] = "86bf21e743f9e459911916e8f732e86d"
	nifiProperties["nifi.security.keyPasswd"] = "86bf21e743f9e459911916e8f732e86d"
	nifiProperties["nifi.security.truststore"] = "./conf/truststore.p12"
	nifiProperties["nifi.security.truststoreType"] = "PKCS12"
	nifiProperties["nifi.security.truststorePasswd"] = "56e1b852671e334bf69188d296632ef2"
	nifiProperties["nifi.security.user.authorizer"] = "single-user-authorizer"
	nifiProperties["nifi.security.allow.anonymous.authentication"] = "false"
	nifiProperties["nifi.security.user.login.identity.provider"] = "single-user-provider"
	nifiProperties["nifi.security.user.jws.key.rotation.period"] = "PT1H"
	nifiProperties["nifi.security.ocsp.responder.url"] = ""
	nifiProperties["nifi.security.ocsp.responder.certificate"] = ""
	nifiProperties["nifi.security.user.oidc.discovery.url"] = ""
	nifiProperties["nifi.security.user.oidc.connect.timeout"] = "5 secs"
	nifiProperties["nifi.security.user.oidc.read.timeout"] = "5 secs"
	nifiProperties["nifi.security.user.oidc.client.id"] = ""
	nifiProperties["nifi.security.user.oidc.client.secret"] = ""
	nifiProperties["nifi.security.user.oidc.preferred.jwsalgorithm"] = ""
	nifiProperties["nifi.security.user.oidc.additional.scopes"] = ""
	nifiProperties["nifi.security.user.oidc.claim.identifying.user"] = ""
	nifiProperties["nifi.security.user.oidc.fallback.claims.identifying.user"] = ""
	nifiProperties["nifi.security.user.oidc.truststore.strategy"] = "JDK"
	nifiProperties["nifi.security.user.knox.url"] = ""
	nifiProperties["nifi.security.user.knox.publicKey"] = ""
	nifiProperties["nifi.security.user.knox.cookieName"] = "hadoop-jwt"
	nifiProperties["nifi.security.user.knox.audiences"] = ""
	nifiProperties["nifi.security.user.saml.idp.metadata.url"] = ""
	nifiProperties["nifi.security.user.saml.sp.entity.id"] = ""
	nifiProperties["nifi.security.user.saml.identity.attribute.name"] = ""
	nifiProperties["nifi.security.user.saml.group.attribute.name"] = ""
	nifiProperties["nifi.security.user.saml.metadata.signing.enabled"] = "falsenifi.cluster.flow.election.max.candidates="
	nifiProperties["nifi.cluster.load.balance.host"] = ""
	nifiProperties["nifi.cluster.load.balance.port"] = "6342"
	nifiProperties["nifi.nar.library.provider.nifi-registry.implementation"] = "org.apache.nifi.registry.extension.NiFiRegistryNarProvider"
	nifiProperties["nifi.security.user.saml.request.signing.enabled"] = "false"
	nifiProperties["nifi.security.user.saml.want.assertions.signed"] = "true"
	nifiProperties["nifi.security.user.saml.signature.algorithm"] = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
	nifiProperties["nifi.security.user.saml.signature.digest.algorithm"] = "http://www.w3.org/2001/04/xmlenc#sha256"
	nifiProperties["nifi.security.user.saml.message.logging.enabled"] = "false"
	nifiProperties["nifi.security.user.saml.authentication.expiration"] = "12 hours"
	nifiProperties["nifi.security.user.saml.single.logout.enabled"] = "false"
	nifiProperties["nifi.security.user.saml.http.client.truststore.strategy"] = "JDK"
	nifiProperties["nifi.security.user.saml.http.client.connect.timeout"] = "30 secs"
	nifiProperties["nifi.security.user.saml.http.client.read.timeout"] = "30 secs"
	nifiProperties["nifi.listener.bootstrap.port"] = "0"
	nifiProperties["nifi.cluster.protocol.heartbeat.interval"] = "5 sec"
	nifiProperties["nifi.cluster.protocol.heartbeat.missable.max"] = "8"
	nifiProperties["nifi.cluster.protocol.is.secure"] = "true"
	nifiProperties["nifi.cluster.is.node"] = "false"
	nifiProperties["nifi.cluster.node.address"] = "nifi-sample-0"
	nifiProperties["nifi.cluster.node.protocol.port"] = ""
	nifiProperties["nifi.cluster.node.protocol.max.threads"] = "50"
	nifiProperties["nifi.cluster.node.event.history.size"] = "25"
	nifiProperties["nifi.cluster.node.connection.timeout"] = "5 sec"
	nifiProperties["nifi.cluster.node.read.timeout"] = "5 sec"
	nifiProperties["nifi.cluster.node.max.concurrent.requests"] = "100"
	nifiProperties["nifi.cluster.firewall.file"] = ""
	nifiProperties["nifi.cluster.flow.election.max.wait.time"] = "5 mins"
	nifiProperties["nifi.cluster.flow.election.max.candidates"] = ""
	nifiProperties["nifi.cluster.load.balance.host"] = ""
	nifiProperties["nifi.cluster.load.balance.port"] = "6342"
	nifiProperties["nifi.cluster.load.balance.connections.per.node"] = "1"
	nifiProperties["nifi.cluster.load.balance.max.thread.count"] = "8"
	nifiProperties["nifi.cluster.load.balance.comms.timeout"] = "30 sec"
	nifiProperties["nifi.zookeeper.connect.string"] = ""
	nifiProperties["nifi.zookeeper.connect.timeout"] = "10 secs"
	nifiProperties["nifi.zookeeper.session.timeout"] = "10 secs"
	nifiProperties["nifi.zookeeper.root.node"] = "/nifi"
	nifiProperties["nifi.zookeeper.client.secure"] = "false"
	nifiProperties["nifi.zookeeper.security.keystore"] = ""
	nifiProperties["nifi.zookeeper.security.keystoreType"] = ""
	nifiProperties["nifi.zookeeper.security.keystorePasswd"] = ""
	nifiProperties["nifi.zookeeper.security.truststore"] = ""
	nifiProperties["nifi.zookeeper.security.truststoreType"] = ""
	nifiProperties["nifi.zookeeper.security.truststorePasswd"] = ""
	nifiProperties["nifi.zookeeper.jute.maxbuffer"] = ""
	nifiProperties["nifi.zookeeper.auth.type"] = ""
	nifiProperties["nifi.zookeeper.kerberos.removeHostFromPrincipal"] = ""
	nifiProperties["nifi.zookeeper.kerberos.removeRealmFromPrincipal"] = ""
	nifiProperties["nifi.kerberos.krb5.file"] = ""
	nifiProperties["nifi.kerberos.service.principal"] = ""
	nifiProperties["nifi.kerberos.service.keytab.location"] = ""
	nifiProperties["nifi.kerberos.spnego.principal"] = ""
	nifiProperties["nifi.kerberos.spnego.keytab.location"] = ""
	nifiProperties["nifi.kerberos.spnego.authentication.expiration"] = "12 hours"
	nifiProperties["nifi.variable.registry.properties"] = ""
	nifiProperties["nifi.analytics.predict.enabled"] = "false"
	nifiProperties["nifi.analytics.predict.interval"] = "3 mins"
	nifiProperties["nifi.analytics.query.interval"] = "5 mins"
	nifiProperties["nifi.analytics.connection.model.implementation"] = "org.apache.nifi.controller.status.analytics.models.OrdinaryLeastSquares"
	nifiProperties["nifi.analytics.connection.model.score.name"] = "rSquared"
	nifiProperties["nifi.analytics.connection.model.score.threshold"] = ".90"
	nifiProperties["nifi.monitor.long.running.task.schedule"] = ""
	nifiProperties["nifi.monitor.long.running.task.threshold"] = ""
	nifiProperties["nifi.diagnostics.on.shutdown.enabled"] = "false"
	nifiProperties["nifi.diagnostics.on.shutdown.verbose"] = "false"
	nifiProperties["nifi.diagnostics.on.shutdown.directory"] = "./diagnostics"
	nifiProperties["nifi.diagnostics.on.shutdown.max.filecount"] = "10"
	nifiProperties["nifi.diagnostics.on.shutdown.max.directory.size"] = "10 MB"
	nifiProperties["nifi.nar.library.provider.hdfs.implementation"] = "org.apache.nifi.nar.hadoop.HDFSNarProvider"
	nifiProperties["nifi.nar.library.provider.hdfs.resources"] = "/path/to/core-site.xml,/path/to/hdfs-site.xml"
	nifiProperties["nifi.nar.library.provider.nifi-registry.implementation"] = "org.apache.nifi.registry.extension.NiFiRegistryNarProvider"

	return &nifiProperties
}

func getNifiProperties(nifi *bigdatav1alpha1.Nifi) *map[string]string {
	nifiConf := *getDefaultNifiProperties()

	// Disable embedded Zookeeper if Nifi has only one instance
	if nifi.Spec.Size == 1 {
		nifiConf["nifi.state.management.embedded.zookeeper.start"] = "false"
	} else {
		nifiConf["nifi.state.management.embedded.zookeeper.start"] = "true"
	}

	return &nifiConf
}

func newConfigMapNifiProperties(nifi *bigdatav1alpha1.Nifi) *corev1.ConfigMap {
	cm := newConfigMapWithName(nifi.Name+nifiPropertiesConfigMapNameSuffix, nifi)

	nifiConf := getNifiProperties(nifi)
	strConf := ""

	for key, value := range *nifiConf {
		strConf += key + "=" + value + "\n"
	}

	cm.Data = map[string]string{
		"nifi.properties": strConf,
	}

	return cm
}

func (r *Reconciler) reconcileConfigMaps(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi) error {
	if err := r.reconcileNifiProperties(ctx, req, nifi); err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) reconcileNifiProperties(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi) error {
	cm := newConfigMapNifiProperties(nifi)

	existingCM := &corev1.ConfigMap{}
	if nifiutils.IsObjectFound(r.Client, nifi.Namespace, cm.Name, existingCM) {
		if !reflect.DeepEqual(cm.Data, existingCM.Data) {
			existingCM.Data = cm.Data
			return r.Client.Update(ctx, existingCM)
		}
		return nil
	}

	// Set Nifi instance as the owner and controller
	if err := ctrl.SetControllerReference(nifi, cm, r.Scheme); err != nil {
		return err
	}
	return r.Client.Create(ctx, cm)
}
