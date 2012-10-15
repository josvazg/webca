package webca

const (
	//
	//
	//            HTML Templates
	//
	//
	htmlTemplates = `{{define "setuphtmlheader"}}
<html>
<head>
<title>WebCA (Setup)</title>
<style type="text/css">
{{template "style.css"}}
</style>
</head>
<body>
<div class="topbar">
<table>
<tr>
<td>
<h1>
<img height="80px" src="/img/CASeal.png"/>
</h1>
</td>
<td class="titleCell">
<h1>
WebCA Setup:
<label class="activated" id="Step1">
<label class="bigger">1</label>
<label class="explanation">{{tr "First User & Mailer Configuration"}}</label>
</label>
<label class="shadowed" id="Step2">
<label class="bigger">2</label>
<label class="explanation">{{tr "Certificate Authority"}}</label>
</label>
<label class="shadowed" id="Step3">
<label class="bigger">3</label>
<label class="explanation">{{tr "WebCA's Server Certificate"}}</label>
</label>
</h1>
</td>
</tr>
</table>
</div>
<script type="text/javascript">
{{template "JSGetID"}}
{{template "JSEvents"}}
{{template "JSFiltering"}}
{{template "JSSetupNavigation"}}
{{template "JSToggleOps"}}
{{template "JSCheckpasswd"}}
</script>
{{end}}

{{define "htmlheader"}}
<div class="topbar">
<h1><img height="80px" src="/img/CASeal.png"/>WebCA</h1>
<style type="text/css">
{{template "style.css"}}
</style>
  <div class="loggedUser">
{{if .LoggedUser}} Logged as: {{.LoggedUser.Fullname}} (<a href="/logout">logout</a>)
{{end}}
  </div>
</div>
<script type="text/javascript">
{{template "JSGetID"}}
</script>
{{end}}


{{define "htmlfooter"}}
<div class="footer">
	<a href="http://github.com/josvazg/webca">Hosted on GitHub</a><br/>
	<a rel="license" href="http://creativecommons.org/licenses/by/3.0/"><img 
       alt="Licencia Creative Commons" style="border-width:0" src="img/ccby.png" />
    </a><br /><a rel="license" href="http://creativecommons.org/licenses/by/3.0/">
    Creative Commons Attribution 3.0 License</a>.
</div>
</body>
</html>
{{end}}

{{define "userDetails"}}
<tr><td class="mainlabel">{{tr "Username"}}:</td>
    <td class="mainlabel">
    <input type="text" class="main" id="Username" name="Username" 
           value="{{.U.Username}}" maxlength="32" onblur="fixUsername(this)"></td></tr>
<tr><td class="label">{{tr "Fullname"}}:</td>
    <td class="label">
    <input type="text" name="Fullname" size="64"  maxlength="64" 
           value="{{.U.Fullname}}"></td></tr>
<tr><td class="label">{{tr "Password"}}:</td>
    <td class="label"><input type="password" id="Password" name="Password" 
        onkeyup="checkPassword(this)"></td>
</tr>
<tr><td class="label">{{tr "Repeat Password"}}:</td>
    <td class="label"><input type="password" id="Password2" name="Password2" 
        onkeyup="checkPassword(this)"></td>
</tr>
<tr><td class="label">{{tr "Email"}}:</td>
    <td class="label"><input type="text" id="Email" name="Email" value="{{.U.Email}}"></td></tr>
{{end}}

{{define "certCommonFields"}}
<tr class="ops"><td class="label">{{tr "Street"}}:</td>
    <td><input type="text" name="{{.Prfx}}.StreetAddress" id="{{.Prfx}}.StreetAddress"  
                           value="{{indexOf .Crt.Name.StreetAddress 0}}"></td></tr>
<tr class="ops"><td class="label">{{tr "Postal Code"}}:</td>
    <td><input type="text" name="{{.Prfx}}.PostalCode" id="{{.Prfx}}.PostalCode"  
                           value="{{indexOf .Crt.Name.PostalCode 0}}"></td></tr>
<tr class="ops"><td class="label">{{tr "Locality"}}:</td>
    <td><input type="text" name="{{.Prfx}}.Locality" id="{{.Prfx}}.Locality" 
                           value="{{indexOf .Crt.Name.Locality 0}}"></td></tr>
<tr class="ops"><td class="label">{{tr "Province"}}:</td>
    <td><input type="text" name="{{.Prfx}}.Province" id="{{.Prfx}}.Province"  
                           value="{{indexOf .Crt.Name.Province 0}}"></td></tr>
<tr class="ops"><td class="label">{{tr "Org. Unit"}}:</td>
    <td><input type="text" name="{{.Prfx}}.OrganizationalUnit" id="{{.Prfx}}.OrganizationalUnit"  
                           value="{{indexOf .Crt.Name.OrganizationalUnit 0}}"></td></tr>
<tr class="ops"><td class="label">{{tr "Organization"}}:</td>
    <td><input type="text" name="{{.Prfx}}.Organization" id="{{.Prfx}}.Organization"
                           value="{{indexOf .Crt.Name.Organization 0}}"></td></tr>
<tr class="ops"><td class="label">{{tr "Country"}}:</td>
    <td><input type="text" name="{{.Prfx}}.Country" id="{{.Prfx}}.Country"
                           value="{{indexOf .Crt.Name.Country 0}}"></td></tr>
<tr class="ops"><td class="label">{{tr "Duration in Days"}}:</td>
    <td><select id="{{.Prfx}}.Duration" name="{{.Prfx}}.Duration">
            <option value='30' 
                    {{if .IsSelected 30}}selected="selected"{{end}}>{{tr "1 Month"}}</option>
            <option value='60' 
                    {{if .IsSelected 60}}selected="selected"{{end}}>{{tr "2 Months"}}</option>
            <option value='90' 
                    {{if .IsSelected 90}}selected="selected"{{end}}>{{tr "3 Months"}}</option>
            <option value='180' 
                    {{if .IsSelected 180}}selected="selected"{{end}}>{{tr "6 Months"}}</option>
            <option value='365' 
                    {{if .IsSelected 365}}selected="selected"{{end}}>{{tr "1 Year"}}</option>
            <option value='730' 
                    {{if .IsSelected 750}}selected="selected"{{end}}>{{tr "2 Years"}}</option>
            <option value='1095' 
                    {{if .IsSelected 1095}}selected="selected"{{end}}>{{tr "3 Years"}}</option>
            <option value='1825' 
                    {{if .IsSelected 1825}}selected="selected"{{end}}>{{tr "5 Years"}}</option>
            <option value='3650' 
                    {{if .IsSelected 3650}}selected="selected"{{end}}>{{tr "10 Years"}}</option>
	</select></td></tr>
{{end}}

{{define "mailerDetails"}}
<tr><td class="label">{{tr "Email"}}:</td>
    <td class="label"><input type="text" id="M.User" name="M.User" value="{{.M.User}}"></td></tr>
<tr><td class="label">{{tr "Email Server"}}:</td>
    <td class="label"><input type="text" name="M.Server" value="{{.Server}}">:<input 
        type="text" name="M.Port" size="6" value="{{.Port}}"></td></tr>
<tr><td class="label">{{tr "Email Password"}}:</td>
    <td class="label"><input type="password" id="M.Password" name="M.Password" 
        onkeyup="checkPassword(this)"></td></tr>
<tr><td class="label">{{tr "Repeat Password"}}:</td>
    <td class="label"><input type="password" id="M.Password2" name="M.Password2" 
        onkeyup="checkPassword(this)"></td></tr>
{{end}}

{{define "certNode"}}
<div class="indent">
{{range .}}
<span class="Cert"><a 
href="/edit?cert={{qEsc .Crt.Subject.CommonName}}">{{.Crt.Subject.CommonName}}</a></span>
<span class="period">{{showPeriod .Crt}}</span>
{{template "certNode" .Childs}}
{{end}}
</div>
{{end}}
`

	//
	//
	//            Javascript Templates
	//
	//
	jsTemplates = `{{define "JSGetID"}}
function $(id) {
	return document.getElementById(id);
}
{{end}}
{{define "JSEvents"}}
function addEvent (x,y,z) { 
	if (document.addEventListener){ 
		x.addEventListener(y,z,false);
	} else { 
		x.attachEvent('on'+y,z); 
	}
}
{{end}}
{{define "JSFiltering"}}
function fixUsername(el) {
	lowercase(el)
	el.value=el.value.replace(/[&\<\>\$#]+/g,'');
	el.value=el.value.replace(" ",'');
}
function lowercase(el) {
	if (el.value!=null) {
		el.value=el.value.toLowerCase()
	}
	return el
}
{{end}}

{{define "JSByClass"}}
function getElementsByClass( searchClass, domNode, tagName) { 
	if (domNode == null) domNode = document;
	if (tagName == null) tagName = '*';
	var el = new Array();
	var tags = domNode.getElementsByTagName(tagName);
	var tcl = " "+searchClass+" ";
	for(i=0,j=0; i<tags.length; i++) { 
		var test = " " + tags[i].className + " ";
		if (test.indexOf(tcl) != -1) 
			el[j++] = tags[i];
	} 
	return el;
} 
{{end}}

{{define "JSSetupNavigation"}}
	var step=1;
	function next() {
		$('Step'+step).className='shadowed';
		$('form'+step).style.display='none';
		step++;
		helper();
		$('Step'+step).className='activated';
		$('form'+step).style.display='';
	}
	function prev() {
		$('Step'+step).className='shadowed';
		$('form'+step).style.display='none';
		step--;
		helper();
		$('Step'+step).className='activated';
		$('form'+step).style.display='';
	}
	function fillMailerConfig() {
		if($('M.User').value==null || $('M.User').value=='') {
			$('M.User').value=$('Email').value;
		}
	}
	function helper() {
		if(step==1) {
			$('Prev').style.visibility='hidden';
		} else {
			$('Prev').style.visibility='';
		}
		if(step==3) {
			fields=["StreetAddress","PostalCode","Locality","Province",
				    "OrganizationalUnit","Organization","Country"];
			for(i=0;i<fields.length;i++) {
				field=fields[i];
				if($('Cert.'+field).value==null || $('Cert.'+field).value=='') {
					$('Cert.'+field).value=$('CA.'+field).value;
				}
			}
			$('Next').style.display='none';
			$('submit').style.display='block';
		} else {
			$('Next').style.display='block';
			$('submit').style.display='none';
		}
	}
	addEvent(window,"load",function(){ $('Email').onblur=fillMailerConfig; });
{{end}}

{{define "JSSetupDone"}}
	for (i=2;i<=4;i++) {
		$('Step'+i).className='activated';
	}
{{end}}

{{define "JSToggleOps"}}
{{template "JSByClass"}}
function toggleOps() {
	el=$('form3')
	trs=getElementsByClass("ops",el,"tr");
	for (i=0;i<trs.length;i++) {
		tr=trs[i]
		if (tr.style.display=='none') {
			tr.style.display='';
			$('toggler').innerHTML='{{tr "Less"}}...';
		} else {
			tr.style.display='none';
			$('toggler').innerHTML='{{tr "More"}}...';
		}
	}
}
addEvent(window,"load",toggleOps);
{{end}}

{{define "JSCheckpasswd"}}
function showError(msg) {
	$('noticeText').innerHTML=msg;
	$('notice').style.visibility='';
	$('submit').disabled=true;
}
function hideError() {
	$('notice').style.visibility='hidden';
	$('submit').disabled=false;
}
function checkPassword(el) {
	if (el.value=="") {
		showError('{{tr "Type some password!"}}');
		return
	}
	oid=el.id;
	if(oid.substr(-1) === "2") {
		oid=oid.substr(0,oid.length-1);
	} else {
		oid=oid+"2";
	}
	if ($(oid).value!=el.value) {
		showError('{{tr "Passwords don't match!"}}');
		return
	}
	hideError();
}
{{end}}`

	//
	//
	//            Page Templates
	//
	//
	pages = `{{define "setup"}}
{{template "setuphtmlheader" .}}
<form action="/setup" method="post">
<table style="width: 100%; height: 500px">
<tr>
<td class="huge">
<a class="huge" id="Prev" style="visibility: hidden" href="javascript:" onclick="prev()">&lt;</a>
</td>
<td style="vertical-align: top">
<div class="notice" style="visibility: hidden" id="notice">
<label class="notice" id="noticeText"><label>
</div>
<div id="form1">
<h2>{{tr "First User & Mailer Configuration"}}</h2>
<div class="explanation">
{{tr "You'll need a user and a password in order to use this application."}} <br/>
</div>
<table class="form">
{{template "userDetails" .}}
</table>
<div class="explanation">
{{tr "If you want to get email notifications before your certificates expires,"}}<p/>
{{tr "we need to configure a sending email account"}}
</div>
<table class="form">
{{template "mailerDetails" .}}
</table>
</div>
<div id="form2" style="display: none">
<h2>{{tr "Certificate Authority"}}</h2>
<div class="explanation">
{{tr "We cannot run our own Web CA on an unsecure http:// connection like this!"}}<p>
{{tr "Lets create the certificates right now... First the Certificate Authority"}}
</div>
<table class="form">
<tr><td class="mainlabel">{{tr "CA Name"}}:</td>
    <td><input type="text" class="main" name="CA.CommonName" 
                                        value="{{.CA.Name.CommonName}}"></td></tr>
{{.LoadCrt .CA "CA" 1095}}
{{template "certCommonFields" .}}
</table>
</div>
<div id="form3" style="display: none">
<h2>{{tr "WebCA's Server Certificate"}}</h2>
<div class="explanation">
{{tr "We now need a certificate for the WebCA server itself..."}}
</div>
<table class="form">
<tr><td class="mainlabel">{{tr "Certificate Name"}}:</td>
    <td><input type="text" class="main" name="Cert.CommonName" 
                                        value="{{.Cert.Name.CommonName}}"></td>
</tr>
<tr><td colspan="2">
<a id="toggler" onclick="toggleOps()" class="control">{{tr "More"}}...</a>
</td></tr>
{{.LoadCrt .Cert "Cert" 365}}
{{template "certCommonFields" .}}
</table>
</div>
</td>
<td class="huge">
<a class="huge" id="Next" href="javascript:" onclick="next()">&gt;</a>
<input style="display: none" type="submit" disabled="true" id="submit" 
       name="submit" value="{{tr "Save"}}">
</td>
</tr>
</table>
</form>
{{template "htmlfooter"}}
{{end}}



{{define "restart"}}
{{template "setuphtmlheader" .}}
<h2>{{.Message}}</h2>
<div class="mediumExplanation" id="text">
{{tr "You'll need to install the CA certificate."}} <p/>
<a href="crt/{{.CAName}}.pem">{{tr "Download CA certificate here"}}</a><p/>
{{tr "In case something goes wrong with the download the file you are looking for is"}}: 
<b>{{.CAName}}.pem</b> <p/>
<p/>
{{tr "Once you are done, you can start using your WebCA right away..."}} <p/>
<a href="https://{{.WebCAURL}}">{{tr "Click here to go into your WebCA"}}</a><p/>
</div>
<script type="text/javascript">
{{template "JSSetupDone"}}
</script>
{{template "htmlfooter"}}
{{end}}


{{define "login"}}
{{template "htmlheader" .}}

<h2>{{tr "WebCA's Login"}}</h2>
{{if .Error}}
<div class="notice" id="notice">
<label class="notice" id="noticeText">{{.Error}}<label>
</div>
{{end}}
<form action="/login" method="post">
<input type="hidden" id="_SESSION_ID" name="_SESSION_ID" value="{{._SESSION_ID}}"/>
<input type="hidden" id="URL" name="URL" value="{{.URL}}"/>
<table class="form">
<tr><td class="label">{{tr "Username"}}:</td>
    <td><input type="text" class="main" name="Username" value="{{.Username}}">
    </td></tr>
<tr><td class="label">{{tr "Password"}}:</td>
    <td><input type="password" class="main" name="Password" value="{{.Password}}">
    </td></tr>
</tr>
<td class="label" colspan="2" style="text-align: center">
<input type="submit" id="submit" name="submit" value='{{tr "Login"}}'>
</td>
</tr>
</table>
</form>

{{template "htmlfooter"}}
{{end}}


{{define "index"}}
{{template "htmlheader" .}}
<h2>{{tr "WebCA's Index"}}</h2>
<div class="data">
<div class="CATitle">{{tr "Local CAs:"}}</div>
{{range .CAs}}
<span class="CA">
<a href="/edit?cert={{qEsc .Crt.Subject.CommonName}}">{{.Crt.Subject.CommonName}}</a>
</span>
<span class="period">{{showPeriod .Crt}}</span></span>
{{template "certNode" .Childs}}
<div class="Cert"><a href="/new?parent={{qEsc .Crt.Subject.CommonName}}"
     >+ {{tr "Add more Certificates to %s..." .Crt.Subject.CommonName}}</a></div>
{{end}}
<p/>
<div class="CA"><a href="/new">+ {{tr "Add more CAs..."}}</a></div>
<div class="CATitle">{{tr "Externally Managed Certificates:"}}</div>
{{range .Others}}
<span class="CA"><a href="/edit">{{.Crt.Subject.CommonName}}</a></span>
<span class="period">{{showPeriod .Crt}}</span>
{{template "certNode" .Childs}}
{{end}}
<div class="CA"><a href="/import">+ {{tr "Import more..."}}</a></div>
</div>
{{template "htmlfooter"}}
{{end}}

{{define "ca"}}
{{template "htmlheader" .}}
<h2>{{.Title}}</h2>
<table class="form">
<tr><td class="mainlabel">{{tr "CA Name"}}:</td>
    <td><input type="text" class="main" name="CA.CommonName" 
                                        value="{{.CA.Name.CommonName}}"></td></tr>
{{.LoadCrt .CA "CA" 1095}}
{{template "certCommonFields" .}}
</table>
{{template "htmlfooter"}}
{{end}}

{{define "cert"}}
{{template "htmlheader" .}}
<h2>{{.Title}}</h2>
<table class="form">
<tr><td class="mainlabel">{{tr "Certificate Name"}}:</td>
    <td><input type="text" class="main" name="Cert.CommonName" 
                                        value="{{.Cert.Name.CommonName}}"></td>
</tr>
{{.LoadCrt .Cert "Cert" 365}}
{{template "certCommonFields" .}}
</table>
{{template "htmlfooter"}}
{{end}}
`
)
