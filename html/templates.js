{{define "JSGetID"}}
function $(id) {
	return document.getElementById(id);
}
function addEvent (x,y,z) { 
	if (document.addEventListener){ 
		x.addEventListener(y,z,false);
	} else { 
		x.attachEvent('on'+y,z); 
	}
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
			$('Next').style.visibility='hidden';
		} else {
			$('Next').style.visibility='';
		}
	}
	addEvent(window,"onload",function(){ $('Email').onblur=fillMailerConfig; });
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
addEvent(window,"onload",toggleOps);
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
{{end}}
